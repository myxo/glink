#pragma once

#include <map>
#include <set>
#include <string>
#include <vector>

struct Message {
    std::string text;
    int index{-1};
    // time
};

struct CidInfo {
    std::string cid;
    std::string name;
    // prev known enppoints?

    friend auto operator<=>(const CidInfo&, const CidInfo&) = default;
};

class Database {
public:
    void AddMessage(std::string_view cid, Message msg);
    void AddCid(CidInfo info);

    std::vector<std::string> GetKnownCids() const;
    std::vector<Message> GetLastMessages(std::string_view cid, size_t count = 1) const;

private:
    std::set<CidInfo> cid_info_;
    std::map<std::string, std::vector<Message>, std::less<>> messages_;
};
