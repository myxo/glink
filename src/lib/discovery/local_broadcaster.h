#pragma once

#include <memory>
#include <string>
#include <chrono>

constexpr uint16_t kBroadcastPort = 9078;

struct BroadcastData {
    std::string id;
    std::string ip;
    uint16_t port;
};

class ILocalBroadcaster {
public:
    virtual void SetBroadcastData(BroadcastData data) = 0;
    virtual void Stop() = 0;
};

std::shared_ptr<ILocalBroadcaster> CreateLocalBroadcaster(std::chrono::milliseconds period);