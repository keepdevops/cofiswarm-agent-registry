#pragma once
// Inline helpers for handle_mode_agents_put.
// Included only by coordinator_routes_modes_put.cpp.

#include "coordinator_routes_includes.h"
#include <set>
#include <string>

namespace modes_put_impl {

struct NormalizedAgents {
    json normalized = json::array();
    json unknown    = json::array();
    size_t requested_count = 0;
};

inline NormalizedAgents normalize_agents(const json& body,
                                          const std::set<std::string>& active_names,
                                          const std::string& mode_name) {
    NormalizedAgents r;
    if (!body.contains("agents") || !body["agents"].is_array()) return r;
    std::set<std::string> seen;
    for (const auto& item : body["agents"]) {
        if (!item.is_string()) continue;
        ++r.requested_count;
        const std::string n = item.get<std::string>();
        if (active_names.count(n)) {
            if (mode_name == "pipeline" || seen.insert(n).second)
                r.normalized.push_back(n);
        } else {
            r.unknown.push_back(n);
        }
    }
    return r;
}

inline json build_response(const json& body, const std::string& mode_name,
                            const json& normalized, const json& unknown,
                            const std::string& unknown_synth_name,
                            int max_select_val, bool has_max, bool has_agents,
                            bool has_variant, bool has_preset, bool has_synth_policy,
                            bool has_classifier_policy, bool has_stage_context,
                            bool has_order, bool persisted,
                            const std::set<std::string>& active_names) {
    json out = {
        {"mode", mode_name}, {"agents", normalized}, {"unknown", unknown},
        {"unknown_synthesizer", unknown_synth_name.empty() ? json(nullptr) : json(unknown_synth_name)},
        {"persisted", persisted}
    };
    if (has_max)               out["max_select"]          = max_select_val;
    if (has_variant)           out["variant_policy"]      = body["variant_policy"];
    if (has_preset)            out["preset"]              = body["preset"];
    if (has_synth_policy)      out["synthesis_policy"]    = body["synthesis_policy"];
    if (has_classifier_policy) out["classifier_policy"]   = body["classifier_policy"];
    if (has_stage_context)     out["stage_context_chars"] = body["stage_context_chars"];
    if (has_order && mode_name == "pipeline") {
        if (body["order"].is_null()) {
            out["order"] = nullptr;
        } else {
            json norm_order = json::array();
            json unk_order  = json::array();
            for (const auto& item : body["order"]) {
                if (!item.is_string()) continue;
                const std::string n = item.get<std::string>();
                if (active_names.count(n)) norm_order.push_back(n);
                else unk_order.push_back(n);
            }
            out["order"] = norm_order;
            if (!unk_order.empty()) out["unknown_order"] = unk_order;
        }
    }
    return out;
}

} // namespace modes_put_impl
