#include "discovery_service.h"

#include "local_broadcaster.h"
#include "local_receiver.h"

class Discovery : public IDiscovery {
public:
    Discovery(net::io_context& io_context) {
        local_receiver_ = CreateLocalReceiver(io_context);
        local_broadcaster_ = CreateLocalBroadcaster(std::chrono::seconds(1), io_context);
    }

    std::map<std::string, Endpoint> GetKnownEndpoints() const override {
        return local_receiver_->GetEndpoints();
    }

    void OnNewEndpoint(std::function<void(std::string, Endpoint)> callback) override {
        local_receiver_->OnNewEndpoint(std::move(callback));
    }

    void SetBroadcastData(BroadcastData data) override {
        local_broadcaster_->SetBroadcastData(std::move(data));
    }

private:
    std::shared_ptr<ILocalReceiver> local_receiver_;
    std::shared_ptr<ILocalBroadcaster> local_broadcaster_;
};

std::shared_ptr<IDiscovery> CreateDiscoveryService(net::io_context& io_context) {
    return std::make_shared<Discovery>(io_context);
}