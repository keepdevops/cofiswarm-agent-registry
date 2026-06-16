#pragma once

#include "json.hpp"
#include <functional>
#include <string>

using json = nlohmann::json;

bool read_swarm_config_doc(const std::string& path, json& doc);

struct SwarmPaths;

bool agent_name_in_persisted_roster(const SwarmPaths& paths, const std::string& name);

void swarm_config_roster_cache_invalidate(const std::string& path);

bool swarm_config_upsert_agent_json_field(const std::string& path, const std::string& name,
    const std::function<void(json&)>& apply_or_build_minimal);
