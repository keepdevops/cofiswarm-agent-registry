#include "coordinator_routes_agent_tokens.h"
#include "coordinator_routes_includes.h"
#include "coordinator_routes_internal.h"

void register_coordinator_routes_agent_tokens(httplib::Server& svr, CoordinatorState& st) {
    svr.Put(R"(/api/agents/([A-Za-z0-9_\-]+)/tokens)",
            [&st](const httplib::Request& req, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        const std::string name = req.matches[1];
        if (!agent_name_in_persisted_roster(st.swarm_paths(), name)) {
            res.status = 404;
            res.set_content(json({{"error","unknown agent"},{"name",name}}).dump(),
                            "application/json");
            return;
        }
        Agent* target = nullptr;
        for (auto& a : st.agents) if (a.name == name) { target = &a; break; }
        try {
            auto body = json::parse(req.body);
            const bool has_max = body.contains("max_tokens") && body["max_tokens"].is_number_integer();
            const bool has_ctx = body.contains("context") && body["context"].is_number_integer();
            const bool has_to  = body.contains("read_timeout_secs") && body["read_timeout_secs"].is_number_integer();
            if (!has_max && !has_ctx && !has_to) {
                res.status = 400;
                res.set_content(json({{"error","need integer 'max_tokens', 'context', or 'read_timeout_secs'"}}).dump(),
                                "application/json");
                return;
            }
            int new_max = has_max ? body["max_tokens"].get<int>() : -1;
            int new_ctx = has_ctx ? body["context"].get<int>() : -1;
            int new_to  = has_to  ? body["read_timeout_secs"].get<int>() : -1;
            if (has_max && (new_max < 64 || new_max > 131072)) {
                res.status = 400;
                res.set_content(json({{"error","max_tokens out of range [64,131072]"}}).dump(),
                                "application/json");
                return;
            }
            if (has_ctx && (new_ctx < 512 || new_ctx > 262144)) {
                res.status = 400;
                res.set_content(json({{"error","context out of range [512,262144]"}}).dump(),
                                "application/json");
                return;
            }
            if (has_to && (new_to < 30 || new_to > 7200)) {
                res.status = 400;
                res.set_content(json({{"error","read_timeout_secs out of range [30,7200]"}}).dump(),
                                "application/json");
                return;
            }

            bool auto_bumped_timeout = false;
            if (has_max && !has_to && new_max > 4096) {
                int min_to = new_max / 20 + 30;
                int prior_to = target ? target->read_timeout_secs : 0;
                if (prior_to < min_to) {
                    new_to = min_to;
                    auto_bumped_timeout = true;
                }
            }
            const bool apply_to = has_to || auto_bumped_timeout;

            if (target && has_max) target->max_tokens = new_max;
            if (target && apply_to) target->read_timeout_secs = new_to;

            TokenPersistParams tp;
            tp.has_max = has_max;
            tp.max_tokens = new_max;
            tp.has_ctx = has_ctx;
            tp.context = new_ctx;
            tp.apply_read_timeout = apply_to;
            tp.read_timeout_secs = new_to;
            DualWriteOutcome dwtok = persist_agent_tokens(st.swarm_paths(), name, tp);
            bool active_ok = dwtok.active_ok;
            bool source_ok = dwtok.source_ok;
            std::cout << "🔢 [agents/" << name << "/tokens] ";
            if (has_max) std::cout << "max_tokens=" << new_max << " ";
            if (has_ctx) std::cout << "context=" << new_ctx << " (next deploy) ";
            if (apply_to) std::cout << "read_timeout_secs=" << new_to
                                    << (auto_bumped_timeout ? " (auto)" : "") << " ";
            if (!active_ok) std::cout << " — active write FAILED";
            if (!source_ok) std::cout << " — source write FAILED";
            std::cout << std::endl;
            json resp = {{"name", name}, {"persisted", active_ok && source_ok}};
            if (has_max) resp["max_tokens"] = new_max;
            if (has_ctx) {
                resp["context"] = new_ctx;
                resp["context_pending_redeploy"] = true;
            }
            if (apply_to) {
                resp["read_timeout_secs"] = new_to;
                if (auto_bumped_timeout) resp["read_timeout_auto_bumped"] = true;
            }
            res.set_content(resp.dump(), "application/json");
        } catch (const std::exception& e) {
            std::cerr << "❌ [agents tokens PUT] " << e.what() << std::endl;
            res.status = 400;
            res.set_content(json({{"error", e.what()}}).dump(), "application/json");
        }
    });
}
