#pragma once

#include "network.h"

std::shared_ptr<IConnection> CreateConnection(std::string ip, uint16_t port);