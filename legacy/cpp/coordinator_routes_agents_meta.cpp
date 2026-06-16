#include "coordinator_routes_includes.h"
#include "coordinator_routes_internal.h"
#include "coordinator_routes_agent_tokens.h"
#include "coordinator_routes_agents_desc.h"

void register_coordinator_routes_agents_meta(httplib::Server& svr, CoordinatorState& st) {
    svr.Put(R"(/api/agents/([A-Za-z0-9_\-]+)/prompt)",
            [&st](const httplib::Request& req, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        const std::string name = req.matches[1];
        try {
            auto body = json::parse(req.body);
            if (!body.contains("system_prompt") || !body["system_prompt"].is_string()) {
                res.status = 400;
                res.set_content(json({{"error","missing 'system_prompt' string"}}).dump(),
                                "application/json");
                return;
            }
            if (!agent_name_in_persisted_roster(st.swarm_paths(), name)) {
                res.status = 404;
                res.set_content(json({{"error","unknown agent"},{"name",name}}).dump(),
                                "application/json");
                return;
            }
            const std::string new_prompt = body["system_prompt"].get<std::string>();

            DualWriteOutcome dw = persist_agent_system_prompt(st.swarm_paths(), name, new_prompt);
            bool active_ok = dw.active_ok;
            bool source_ok = dw.source_ok;

            bool live_ok = false;
            for (auto& a : st.agents) if (a.name == name) { a.system_prompt = new_prompt; live_ok = true; break; }

            if (!active_ok && !source_ok && !live_ok) {
                res.status = 500;
                res.set_content(json({{"error","failed to persist prompt"},{"name",name}}).dump(),
                                "application/json");
                return;
            }
            std::cout << "✏️  [agents/" << name << "/prompt] updated ("
                      << new_prompt.size() << " chars)";
            if (!active_ok)  std::cout << " — active write skipped/FAILED";
            if (!st.source_config_path_global.empty() && !source_ok)
                std::cout << " — source write FAILED";
            std::cout << std::endl;
            response_cache::clear();
            res.set_content(json({
                {"name", name},
                {"system_prompt", new_prompt},
                {"persisted", active_ok || source_ok},
                {"live", live_ok}
            }).dump(), "application/json");
        } catch (const std::exception& e) {
            std::cerr << "❌ [agents prompt PUT] " << e.what() << std::endl;
            res.status = 400;
            res.set_content(json({{"error", e.what()}}).dump(), "application/json");
        }
    });

    agents_desc::register_route(svr, st);
    register_coordinator_routes_agent_tokens(svr, st);
}
