#include "coordinator_routes_includes.h"
#include "coordinator_routes_internal.h"
#include "coordinator_routes_modes_put_impl.h"
#include "coordinator_routes_modes_put_apply.h"

void handle_mode_agents_put(CoordinatorState& st,
                             const std::string& mode_name,
                             const httplib::Request& req,
                             httplib::Response& res)
{
    try {
        auto body = json::parse(req.body);
        const bool has_agents  = body.contains("agents") && body["agents"].is_array();
        const bool has_max     = body.contains("max_select") && body["max_select"].is_number_integer();
        const bool has_synth   = body.contains("synthesizer")
                                 && (body["synthesizer"].is_string() || body["synthesizer"].is_null());
        const bool has_variant = body.contains("variant_policy")
                                 && (body["variant_policy"].is_string() || body["variant_policy"].is_null());
        const bool has_preset  = body.contains("preset")
                                 && (body["preset"].is_string() || body["preset"].is_null());
        const bool has_synth_policy = body.contains("synthesis_policy")
                                 && (body["synthesis_policy"].is_string() || body["synthesis_policy"].is_null());
        const bool has_classifier_policy = body.contains("classifier_policy")
                                 && (body["classifier_policy"].is_string() || body["classifier_policy"].is_null());
        const bool has_stage_context = body.contains("stage_context_chars")
                                 && body["stage_context_chars"].is_number_integer();
        const bool has_order   = body.contains("order")
                                 && (body["order"].is_array() || body["order"].is_null());
        if (has_order && mode_name != "pipeline") {
            res.status = 400;
            res.set_content(json({{"error","order is only supported for pipeline mode"}}).dump(),
                            "application/json");
            return;
        }
        if (!has_agents && !has_max && !has_synth && !has_variant && !has_preset
            && !has_synth_policy && !has_classifier_policy && !has_stage_context && !has_order) {
            res.status = 400;
            res.set_content(json({{"error","provide agents, max_select, synthesizer, mode policy, or order"}}).dump(),
                            "application/json");
            return;
        }

        std::set<std::string> active_names;
        for (const auto& a : st.agents) active_names.insert(a.name);

        auto na = modes_put_impl::normalize_agents(body, active_names, mode_name);
        if (has_agents && na.requested_count > 0 && na.normalized.empty()) {
            res.status = 409;
            res.set_content(json({
                {"error","all requested agents unknown — refusing to erase roster"},
                {"mode", mode_name}, {"unknown", na.unknown},
                {"hint","send agents:[] explicitly to clear the override"}
            }).dump(), "application/json");
            return;
        }

        std::string unknown_synth_name;
        int max_select_val = 0;
        if (has_max) {
            max_select_val = body["max_select"].get<int>();
            if (max_select_val < 1) max_select_val = 1;
        }
        bool persisted = false;
        {
            std::lock_guard<std::mutex> lock(st.modes_config_mutex);
            if (!st.modes_config.contains(mode_name) || !st.modes_config[mode_name].is_object())
                st.modes_config[mode_name] = json::object();
            auto& cfg = st.modes_config[mode_name];
            if (has_agents) cfg["agents"] = na.normalized;
            if (has_max)    cfg["max_select"] = max_select_val;
            if (has_synth)  modes_put_apply::apply_synthesizer(body, cfg, active_names, unknown_synth_name);
            modes_put_apply::apply_string_option(body, cfg, "variant_policy",       has_variant);
            modes_put_apply::apply_string_option(body, cfg, "preset",               has_preset);
            modes_put_apply::apply_string_option(body, cfg, "synthesis_policy",     has_synth_policy);
            modes_put_apply::apply_string_option(body, cfg, "classifier_policy",    has_classifier_policy);
            if (has_stage_context) {
                int v = body["stage_context_chars"].get<int>();
                if (v < 0) v = 0;
                cfg["stage_context_chars"] = v;
            }
            if (has_order) modes_put_apply::apply_order(body, cfg, mode_name, active_names);
            persisted = coordinator_persist_modes_locked(st);
        }
        std::cout << "🧩 [modes/" << mode_name << "/agents] "
                  << (has_agents ? std::to_string(na.normalized.size()) + " agent(s) " : "")
                  << (has_max ? "max_select=" + std::to_string(max_select_val) : "")
                  << (persisted ? "" : " (persistence FAILED)") << std::endl;

        json out = modes_put_impl::build_response(
            body, mode_name, na.normalized, na.unknown, unknown_synth_name,
            max_select_val, has_max, has_agents, has_variant, has_preset,
            has_synth_policy, has_classifier_policy, has_stage_context,
            has_order, persisted, active_names);

        if (!has_agents) {
            std::lock_guard<std::mutex> lock(st.modes_config_mutex);
            if (st.modes_config.contains(mode_name)
                && st.modes_config[mode_name].contains("agents")
                && st.modes_config[mode_name]["agents"].is_array()) {
                out["agents"] = st.modes_config[mode_name]["agents"];
            }
        }
        res.set_content(out.dump(), "application/json");
    } catch (const std::exception& e) {
        std::cerr << "❌ [modes/agents PUT] " << e.what() << std::endl;
        res.status = 400;
        res.set_content(json({{"error", e.what()}}).dump(), "application/json");
    }
}
