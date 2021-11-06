#include "engine.h"
#include "discovery_service.h"
#include "utils.h"
#include "server.h"

#include <fmt/format.h>
#include <boost/asio.hpp>
#include "spdlog/spdlog.h"
#include <thread>

#undef SendMessage

namespace net = boost::asio;

class Engine : public IEngine {
public:
    Engine() {

    }

    std::vector<std::string> GetKnownCid() override {
        return {};
    }
    
    void SendMessage(std::string msg, std::string_view to_cid) override {
        // TODO: error handling
        server_->Deliver(msg, to_cid);
        spdlog::debug("Send message to {}", to_cid);

        // spdlog::warn("Trying to send message [{}] to {}, but connection is down", msg, to_cid);
    }

    void OnConnectionRequest(std::function<void(std::string_view cid)> cb) override {
    }

    void SetSelfInfo(std::string cid, std::string name) override {
        own_cid_ = cid;
        own_name_ = name;

        InitConnection(); // TODO: relocate
    }

private:
    void InitConnection() {
        auto local_ips = GetLocalInterfacesIp();

        server_ = std::make_unique<Server>(io_context_, own_cid_);

        discovery_service_ = CreateDiscoveryService(io_context_);
        discovery_service_->SetBroadcastData(
            BroadcastData{.id = own_cid_, .ep{.ip = local_ips.front(), .port = server_->GetPort()}});

        discovery_service_->OnNewEndpoint([this](std::string id, Endpoint ep) {
            if (id == own_cid_) {
                return;
            }
            spdlog::info("Found new endpoint: {}:{}, id:{}", ep.ip, ep.port, id);
            // Options:
            // 1. Make this coroutine
            // 2. Use task based library
            // 3. Use actor / mesaging model
            server_->MakeConnectionTo(ep.ip, ep.port, id);
            // server_->Deliver(fmt::format("Hello from {}!", own_cid_), id);
        });

        thread_ = std::jthread{[this]{ io_context_.run(); }};

    }

private:
    std::shared_ptr<IDiscovery> discovery_service_;
    std::unique_ptr<Server> server_;

    net::io_context io_context_;
    std::jthread thread_;
    std::string own_cid_;
    std::string own_name_;
};

std::shared_ptr<IEngine> CreateEngine() {
    return std::make_shared<Engine>();
}
