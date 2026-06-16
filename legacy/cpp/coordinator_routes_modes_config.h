#pragma once
#include "coordinator_routes_includes.h"
#include "coordinator_routes_internal.h"
#include <set>
#include <string>

namespace modes_config {

inline json build_agents_response(CoordinatorState& st, const std::string& mode_name) {
    json configured = json::array();
    bool explicit_set = false;
    {
        std::lock_guard<std::mutex> lock(st.modes_config_mutex);
        if (st.modes_config.contains(mode_name)
            && st.modes_config[mode_name].contains("agents")
            && st.modes_config[mode_name]["agents"].is_array()
            && !st.modes_config[mode_name]["agents"].empty()) {
            configured = st.modes_config[mode_name]["agents"];
            explicit_set = true;
        }
    }
    json all = json::array();
    std::set<std::string> active_set;
    for (const auto& a : st.agents) { all.push_back(a.name); active_set.insert(a.name); }

    json effective_agents = json::array();
    json stale = json::array();
    if (explicit_set) {
        std::set<std::string> emitted;
        for (const auto& item : configured) {
            if (!item.is_string()) continue;
            const std::string n = item.get<std::string>();
            if (active_set.count(n)) {
                if (mode_name == "pipeline" || emitted.insert(n).second)
                    effective_agents.push_back(n);
            } else {
                stale.push_back(n);
            }
        }
    } else {
        effective_agents = all;
    }

    json out = {
        {"mode", mode_name}, {"agents", effective_agents},
        {"configured_agents", configured}, {"stale", stale},
        {"explicit", explicit_set}, {"available", all}
    };
    {
        std::lock_guard<std::mutex> lock(st.modes_config_mutex);
        if (st.modes_config.contains(mode_name)) {
            const auto& mc = st.modes_config[mode_name];
            if (mc.contains("max_select") && mc["max_select"].is_number_integer())
                out["max_select"] = mc["max_select"];
            if (mc.contains("synthesizer") && mc["synthesizer"].is_string())
                out["synthesizer"] = mc["synthesizer"];
            for (const auto& key : {"variant_policy","preset","synthesis_policy","classifier_policy"}) {
                if (mc.contains(key) && mc[key].is_string()) out[key] = mc[key];
            }
            if (mc.contains("stage_context_chars") && mc["stage_context_chars"].is_number_integer())
                out["stage_context_chars"] = mc["stage_context_chars"];
            if (mode_name == "pipeline" && mc.contains("order") && mc["order"].is_array())
                out["order"] = mc["order"];
        }
    }
    return out;
}

} // namespace modes_config
