#pragma once
// Inline description PUT route — included only by coordinator_routes_agents_meta.cpp.

#include "coordinator_routes_includes.h"
#include "coordinator_routes_internal.h"

namespace agents_desc {

inline void register_route(httplib::Server& svr, CoordinatorState& st) {
    svr.Put(R"(/api/agents/([A-Za-z0-9_\-]+)/description)",
            [&st](const httplib::Request& req, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        const std::string name = req.matches[1];
        try {
            auto body = json::parse(req.body);
            if (!body.contains("description") || !body["description"].is_string()) {
                res.status = 400;
                res.set_content(json({{"error","missing 'description' string"}}).dump(),
                                "application/json");
                return;
            }
            if (!agent_name_in_persisted_roster(st.swarm_paths(), name)) {
                res.status = 404;
                res.set_content(json({{"error","unknown agent"},{"name",name}}).dump(),
                                "application/json");
                return;
            }
            const std::string new_desc = body["description"].get<std::string>();

            DualWriteOutcome dwd = persist_agent_description(st.swarm_paths(), name, new_desc);
            bool active_ok = dwd.active_ok;
            bool source_ok = dwd.source_ok;

            bool live_ok = false;
            for (auto& a : st.agents) if (a.name == name) { a.description = new_desc; live_ok = true; break; }

            if (!active_ok && !source_ok && !live_ok) {
                res.status = 500;
                res.set_content(json({{"error","failed to persist description"},{"name",name}}).dump(),
                                "application/json");
                return;
            }
            std::cout << "✏️  [agents/" << name << "/description] updated ("
                      << new_desc.size() << " chars)";
            if (!active_ok)  std::cout << " — active write skipped/FAILED";
            if (!st.source_config_path_global.empty() && !source_ok)
                std::cout << " — source write FAILED";
            std::cout << std::endl;
            response_cache::clear();
            res.set_content(json({
                {"name", name},
                {"description", new_desc},
                {"persisted", active_ok || source_ok},
                {"live", live_ok}
            }).dump(), "application/json");
        } catch (const std::exception& e) {
            std::cerr << "❌ [agents description PUT] " << e.what() << std::endl;
            res.status = 400;
            res.set_content(json({{"error", e.what()}}).dump(), "application/json");
        }
    });
}

} // namespace agents_desc
