#pragma once

#include "discovery_service.h"

#include <memory>
#include <string>
#include <chrono>

class ILocalBroadcaster {
public:
    virtual void SetBroadcastData(BroadcastData data) = 0;
    virtual void Stop() = 0;
};

std::shared_ptr<ILocalBroadcaster> CreateLocalBroadcaster(std::chrono::milliseconds period);