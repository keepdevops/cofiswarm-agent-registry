#pragma once
#include "swarm_config_store.h"
#include "json.hpp"
#include <filesystem>
#include <fstream>
#include <iostream>
#include <string>

using json = nlohmann::json;
namespace fs = std::filesystem;

namespace roster_io {

inline bool path_identity(const std::string& path,
                           std::uintmax_t& sz,
                           fs::file_time_type& mt) {
    std::error_code ec;
    if (!fs::exists(path, ec)) return false;
    sz = fs::file_size(path, ec);
    mt = fs::last_write_time(path, ec);
    return !ec;
}

inline bool read_doc(const std::string& path, json& doc) {
    std::ifstream in(path);
    if (!in.is_open()) {
        std::cerr << "❌ [persist] cannot read " << path << std::endl;
        return false;
    }
    try {
        doc = json::parse(in);
    } catch (const std::exception& e) {
        std::cerr << "❌ [persist] parse failed (" << path << "): " << e.what() << std::endl;
        return false;
    }
    return true;
}

inline bool upsert_agent_field(const std::string& path, const std::string& name,
    const std::function<void(json&)>& apply_or_build_minimal,
    const std::function<void(const std::string&)>& invalidate_cache) {
    if (path.empty()) return false;
    json doc;
    if (!read_doc(path, doc)) return false;
    if (!doc.contains("agents") || !doc["agents"].is_array())
        doc["agents"] = json::array();
    bool found = false;
    for (auto& a : doc["agents"]) {
        if (a.is_object() && a.value("name", std::string()) == name) {
            apply_or_build_minimal(a);
            found = true;
            break;
        }
    }
    if (!found) {
        json row = {{"name", name}};
        apply_or_build_minimal(row);
        doc["agents"].push_back(row);
    }
    std::ofstream out(path);
    if (!out.is_open()) return false;
    out << doc.dump(2);
    invalidate_cache(path);
    return true;
}

} // namespace roster_io
