#include "coordinator_routes_includes.h"
#include "coordinator_routes_internal.h"

void register_coordinator_routes_core(httplib::Server& svr, CoordinatorState& st) {
    // 1. Health
    svr.Get("/api/health", [&st](const httplib::Request&, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        res.set_content("{\"status\":\"ok\",\"engine\":\"swarm-matrix\"}", "application/json");
    });

    // 2. Agent list
    svr.Get("/api/agents", [&st](const httplib::Request&, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        json list = json::array();
        for (const auto& a : st.agents) {
            json obj = {{"name", a.name}, {"port", a.port}, {"engine", a.engine}};
            if (!a.description.empty()) obj["description"] = a.description;
            if (!a.backend.empty())     obj["backend"]     = a.backend;
            if (!a.model.empty())       obj["model"]       = a.model;
            if (!a.draft_model.empty()) obj["draft_model"] = a.draft_model;
            if (a.draft_max > 0)        obj["draft_max"]   = a.draft_max;
            if (!a.inference_backend.empty()) obj["inference_backend"] = a.inference_backend;
            list.push_back(obj);
        }
        res.set_content(list.dump(), "application/json");
    });

    // 3. History
    svr.Get("/api/history", [&st](const httplib::Request&, httplib::Response& res) {
        res.set_header("Access-Control-Allow-Origin", "*");
        std::lock_guard<std::mutex> lock(st.history_mutex);
        res.set_content(json(st.history).dump(), "application/json");
    });
}
