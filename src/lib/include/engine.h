#pragma once

#include <string>
#include <vector>
#include <functional>
#include <memory>

#include <boost/asio.hpp>
#include "boost/asio/io_context.hpp"

class IEngine {
public:
    virtual std::vector<std::string> GetKnownCid() = 0;
    virtual void SendMessage(std::string meg, std::string_view to_cid) = 0;

    virtual void OnConnectionRequest(std::function<void(std::string_view cid)> cb) = 0;

    virtual void SetSelfInfo(std::string cid, std::string name) = 0; // TODO: different interface
};

std::shared_ptr<IEngine> CreateEngine(boost::asio::io_context& io_context);
