#pragma once

#include "coordinator_context.h"
#include "httplib.h"

void register_coordinator_routes_agent_tokens(httplib::Server& svr, CoordinatorState& st);
