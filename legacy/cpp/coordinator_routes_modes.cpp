#include "coordinator_routes_includes.h"
#include "coordinator_routes_internal.h"
#include "coordinator_routes_modes_config.h"

void register_coordinator_routes_modes(httplib::Server& svr, CoordinatorState& st) {
    svr.Get("/api/modes", [&st](const httplib::Request&, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        const std::string cur = modes::active();
        json out = json::array();
        for (const auto& m : modes::list())
            out.push_back({{"name", m.name}, {"description", m.description}, {"active", m.name == cur}});
        res.set_content(out.dump(), "application/json");
    });

    svr.Get("/api/modes/active", [&st](const httplib::Request&, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        res.set_content(json({{"mode", modes::active()}}).dump(), "application/json");
    });

    svr.Post("/api/modes/active", [&st](const httplib::Request& req, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        try {
            auto j = json::parse(req.body);
            std::string name = j.value("mode", "");
            if (!modes::set_active(name)) {
                json available = json::array();
                for (const auto& m : modes::list()) available.push_back(m.name);
                res.status = 404;
                res.set_content(json({
                    {"error","unknown mode"}, {"requested",name}, {"available",available}
                }).dump(), "application/json");
                return;
            }
            std::cout << "🧠 active mode switched to: " << name << std::endl;
            res.set_content(json({{"mode", name}}).dump(), "application/json");
        } catch (const std::exception& e) {
            std::cerr << "❌ [modes/active] " << e.what() << std::endl;
            res.status = 400;
            res.set_content(json({{"error", e.what()}}).dump(), "application/json");
        }
    });

    // GET /api/modes/<name>/agents — read per-mode roster
    svr.Get(R"(/api/modes/([A-Za-z0-9_-]+)/agents)",
            [&st](const httplib::Request& req, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        const std::string mode_name = req.matches[1];
        if (!modes::get(mode_name)) {
            res.status = 404;
            res.set_content(json({{"error","unknown mode"},{"mode",mode_name}}).dump(),
                            "application/json");
            return;
        }
        res.set_content(modes_config::build_agents_response(st, mode_name).dump(),
                        "application/json");
    });

    // PUT /api/modes/<name>/agents — write per-mode roster (body handled in modes_put.cpp)
    svr.Put(R"(/api/modes/([A-Za-z0-9_-]+)/agents)",
            [&st](const httplib::Request& req, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        const std::string mode_name = req.matches[1];
        if (!modes::get(mode_name)) {
            res.status = 404;
            res.set_content(json({{"error","unknown mode"},{"mode",mode_name}}).dump(),
                            "application/json");
            return;
        }
        handle_mode_agents_put(st, mode_name, req, res);
    });
}
