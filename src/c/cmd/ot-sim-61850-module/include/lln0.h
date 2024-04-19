#ifndef LLN0_H
#define LLN0_H

#include <stdlib.h>
#include <libiec61850/iec61850_model.h>

extern LogicalNode   iedModel_WTG_LLN0;
extern DataObject    iedModel_WTG_LLN0_Mod;
extern DataAttribute iedModel_WTG_LLN0_Mod_stVal;
extern DataAttribute iedModel_WTG_LLN0_Mod_q;
extern DataAttribute iedModel_WTG_LLN0_Mod_t;
extern DataAttribute iedModel_WTG_LLN0_Mod_ctlModel;
extern DataObject    iedModel_WTG_LLN0_Beh;
extern DataAttribute iedModel_WTG_LLN0_Beh_stVal;
extern DataAttribute iedModel_WTG_LLN0_Beh_q;
extern DataAttribute iedModel_WTG_LLN0_Beh_t;
extern DataObject    iedModel_WTG_LLN0_Health;
extern DataAttribute iedModel_WTG_LLN0_Health_stVal;
extern DataAttribute iedModel_WTG_LLN0_Health_q;
extern DataAttribute iedModel_WTG_LLN0_Health_t;
extern DataObject    iedModel_WTG_LLN0_NamPlt;
extern DataAttribute iedModel_WTG_LLN0_NamPlt_vendor;
extern DataAttribute iedModel_WTG_LLN0_NamPlt_swRev;
extern DataAttribute iedModel_WTG_LLN0_NamPlt_configRev;

#endif // LLN0_H