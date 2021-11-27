#include "database.h"

#include <algorithm>
#include <ranges>
#include <cassert>

void Database::AddMessage(std::string_view cid, Message msg) {
    messages_[std::string{cid}].push_back(std::move(msg));
}

void Database::AddCid(CidInfo info) {
    cid_info_.insert(std::move(info));
}

std::vector<std::string> Database::GetKnownCids() const {
    std::vector<std::string> res;
    for (auto const& cid: cid_info_) {
        res.push_back(cid.cid);
    }
    return res;
}

std::vector<Message> Database::GetLastMessages(std::string_view cid, size_t count) const {
    auto it = messages_.find(cid);
    assert(it  != messages_.end());

    std::vector<Message> res;
    for (auto it2 = it->second.rbegin(); it2 != it->second.rend() && count != 0; ++it, count--) {
        res.push_back(*it2);
    }

    return res;
}
