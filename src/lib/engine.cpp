#include "engine.h"
#include "discovery_service.h"
#include "network.h"
#include "utils.h"
#include "client.h"

#include <fmt/format.h>
#include <boost/asio.hpp>

#undef SendMessage

namespace net = boost::asio;

class Engine : public IEngine{
public:
    Engine() {
        connection_pool_ = CreateConnectionPool();
        discovery_service_ = CreateDiscoveryService(io_context_);

        discovery_service_->OnNewEndpoint([this](std::string id, Endpoint ep) {
            if (id == own_cid_) {
                return;
            }
            auto connection = ::CreateConnection(ep.ip, ep.port, io_context_);
            connection_pool_->AddConnection(id, connection);
            connection->SendMesasge(fmt::format("Hello from {}!", own_cid_));
        });

    }

    std::vector<std::string> GetKnownCid() override {
        return {};
    }
    
    void SendMessage(std::string meg, std::string_view to_cid) override {
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

        server_ = std::make_unique<Server>(io_context_);

        discovery_service_->SetBroadcastData(
            BroadcastData{.id = own_cid_, .ep{.ip = local_ips.front(), .port = server_->GetPort()}});

        thread_ = std::jthread{[this]{ io_context_.run(); }};

    }

private:
    std::shared_ptr<IConnectionPool> connection_pool_;
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