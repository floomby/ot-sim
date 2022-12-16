package cpu

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	otsim "github.com/patsec/ot-sim"
)

var hostname string

func init() {
	var err error

	if hostname, err = os.Hostname(); err != nil {
		panic(err)
	}
}

// StartModule starts the process for another module and monitors it to make
// sure it doesn't die, restarting it if it does.
func StartModule(ctx context.Context, name, path string, args ...string) error {
	exePath, err := exec.LookPath(path)
	if err != nil {
		return fmt.Errorf("module executable does not exist at %s", path)
	}

	otsim.Waiter.Add(1)

	go func() {
		defer otsim.Waiter.Done()

		var (
			elasticClient *http.Client
			elasticURL    string
			lokiClient    *http.Client
			lokiURL       string
		)

		if e := ctxGetElasticEndpoint(ctx); e != "" {
			index := ctxGetElasticIndex(ctx)

			elasticClient = new(http.Client)
			elasticURL = fmt.Sprintf("%s/%s/_bulk", e, index)
		}

		if e := ctxGetLokiEndpoint(ctx); e != "" {
			lokiClient = new(http.Client)
			lokiURL = fmt.Sprintf("%s/loki/api/v1/push", e)
		}

		for {
			// Not using `exec.CommandContext` here since we're catching the context
			// being canceled below in order to gracefully terminate the child
			// process. Using `exec.CommandContext` forcefully kills the child process
			// when the context is canceled.
			cmd := exec.Command(exePath, args...)
			cmd.Env = os.Environ()

			stdout, _ := cmd.StdoutPipe()
			stderr, _ := cmd.StderrPipe()

			fmt.Printf("[CPU] starting %s module\n", name)

			if err := cmd.Start(); err != nil {
				fmt.Printf("[CPU] [ERROR] starting %s module: %v\n", name, err)
				return
			}

			go func() {
				scanner := bufio.NewScanner(stdout)
				scanner.Split(bufio.ScanLines)

				var values [][]string

				for scanner.Scan() {
					log := scanner.Text()

					if elasticClient != nil || lokiClient != nil {
						values = append(values, []string{fmt.Sprintf("%d", time.Now().UnixNano()), log})

						if len(values) >= 10 {
							if elasticClient != nil {
								bulk := buildElasticBulk(name, "log", values)

								resp, err := elasticClient.Post(elasticURL, "application/x-ndjson", bytes.NewBuffer(bulk))
								if err == nil {
									resp.Body.Close()
								}
							}

							if lokiClient != nil {
								stream := buildLokiStream(name, "log", values)

								resp, err := lokiClient.Post(lokiURL, "application/json", bytes.NewBuffer(stream))
								if err == nil {
									resp.Body.Close()
								}
							}

							values = nil
						}
					}

					fmt.Printf("[LOG] %s\n", log)
				}
			}()

			go func() {
				scanner := bufio.NewScanner(stderr)
				scanner.Split(bufio.ScanLines)

				var values [][]string

				for scanner.Scan() {
					errLog := scanner.Text()

					if elasticClient != nil || lokiClient != nil {
						values = append(values, []string{fmt.Sprintf("%d", time.Now().UnixNano()), errLog})

						if len(values) >= 10 {
							if elasticClient != nil {
								bulk := buildElasticBulk(name, "error", values)

								resp, err := elasticClient.Post(elasticURL, "application/x-ndjson", bytes.NewBuffer(bulk))
								if err == nil {
									resp.Body.Close()
								}
							}

							if lokiClient != nil {
								stream := buildLokiStream(name, "error", values)

								resp, err := lokiClient.Post(lokiURL, "application/json", bytes.NewBuffer(stream))
								if err == nil {
									resp.Body.Close()
								}
							}

							values = nil
						}
					}

					fmt.Printf("[LOG] [ERROR] %s\n", errLog)
				}
			}()

			wait := make(chan error)

			go func() {
				err := cmd.Wait()
				wait <- err
			}()

			select {
			case err := <-wait:
				fmt.Printf("[CPU] [ERROR] %s module died (%v)... restarting\n", name, err)
				continue
			case <-ctx.Done():
				fmt.Printf("[CPU] stopping %s module\n", name)
				cmd.Process.Signal(syscall.SIGTERM)

				select {
				case <-wait: // SIGTERM *should* cause cmd to exit
					fmt.Printf("[CPU] %s module has stopped\n", name)
					return
				case <-time.After(10 * time.Second):
					fmt.Printf("[CPU] forcefully killing %s module\n", name)
					cmd.Process.Kill()

					return
				}
			}
		}
	}()

	return nil
}

func buildElasticBulk(name, typ string, values [][]string) []byte {
	var docs []string

	for _, val := range values {
		ts, _ := strconv.ParseInt(val[0], 10, 64)
		ts = ts / 1000000 // milliseconds since epoch

		docs = append(docs, `{"index":{}}`)

		doc := fmt.Sprintf(
			`{"@timestamp": %d, "host": %s, "module": %s, "type": %s, "log": %s}`,
			ts, hostname, name, typ, val[1],
		)

		docs = append(docs, doc)
	}

	body := strings.Join(docs, "\n")
	return []byte(body)
}

func buildLokiStream(name, typ string, values [][]string) []byte {
	stream := map[string]any{
		"stream": map[string]any{
			"host":   hostname,
			"module": name,
			"type":   typ,
		},
		"values": values,
	}

	streams := map[string]any{
		"streams": []any{stream},
	}

	body, _ := json.Marshal(streams)
	return body
}
