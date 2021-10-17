#pragma once

#include "network.h"

namespace net = boost::asio;

std::shared_ptr<IConnection> CreateConnection(std::string ip, uint16_t port, net::io_context& io_context);