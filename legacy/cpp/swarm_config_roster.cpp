#include "swarm_config_roster.h"
#include "swarm_config_roster_io.h"

#include <mutex>
#include <unordered_map>
#include <unordered_set>

static std::mutex g_roster_cache_mu;

struct RosterSnapshot {
    std::filesystem::file_time_type mtime{};
    std::uintmax_t size = 0;
    std::unordered_set<std::string> names;
};

static std::unordered_map<std::string, RosterSnapshot> g_roster_cache;

void swarm_config_roster_cache_invalidate(const std::string& path) {
    if (path.empty()) return;
    std::lock_guard<std::mutex> lk(g_roster_cache_mu);
    g_roster_cache.erase(path);
}

bool read_swarm_config_doc(const std::string& path, json& doc) {
    return roster_io::read_doc(path, doc);
}

bool agent_name_in_persisted_roster(const SwarmPaths& paths, const std::string& name) {
    const std::string& roster_path = !paths.source_config_path.empty()
        ? paths.source_config_path
        : paths.active_config_path;
    if (roster_path.empty()) return false;

    std::uintmax_t sz = 0;
    std::filesystem::file_time_type mt{};
    if (!roster_io::path_identity(roster_path, sz, mt)) return false;

    {
        std::lock_guard<std::mutex> lk(g_roster_cache_mu);
        auto it = g_roster_cache.find(roster_path);
        if (it != g_roster_cache.end()
            && it->second.size == sz
            && it->second.mtime == mt) {
            return it->second.names.count(name) > 0;
        }
    }

    json doc;
    if (!roster_io::read_doc(roster_path, doc)) return false;
    if (!doc.contains("agents") || !doc["agents"].is_array()) return false;
    std::uintmax_t sz_after = sz;
    std::filesystem::file_time_type mt_after = mt;
    roster_io::path_identity(roster_path, sz_after, mt_after);
    std::unordered_set<std::string> names;
    bool hit = false;
    for (const auto& a : doc["agents"]) {
        if (!a.is_object()) continue;
        const std::string n = a.value("name", std::string());
        if (!n.empty()) names.insert(n);
        if (n == name) hit = true;
    }
    {
        std::lock_guard<std::mutex> lk(g_roster_cache_mu);
        RosterSnapshot snap;
        snap.mtime = mt_after;
        snap.size = sz_after;
        snap.names = std::move(names);
        g_roster_cache[roster_path] = std::move(snap);
    }
    return hit;
}

bool swarm_config_upsert_agent_json_field(const std::string& path, const std::string& name,
    const std::function<void(json&)>& apply_or_build_minimal) {
    return roster_io::upsert_agent_field(path, name, apply_or_build_minimal,
        swarm_config_roster_cache_invalidate);
}
