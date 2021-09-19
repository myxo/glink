#pragma once

#include <chrono>
#include <memory>
#include <string>

class ILocalReceiver {
public:
    virtual void Stop() = 0;
};

std::shared_ptr<ILocalReceiver> CreateLocalReceiver();