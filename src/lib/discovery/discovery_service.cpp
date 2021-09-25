#include "discovery_service.h"
#include "local_receiver.h"
#include "local_broadcaster.h"


class Discovery : public IDiscovery {
public:
    Discovery() {
        local_receiver_ = CreateLocalReceiver();
        local_broadcaster_ = CreateLocalBroadcaster(std::chrono::seconds(1));
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

std::shared_ptr<IDiscovery> CreateDiscoveryService() {
    return std::make_shared<Discovery>();
}