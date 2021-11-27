#include "engine.h"
#include "database.h"
#include "discovery_service.h"
#include "utils.h"
#include "server.h"
#include "message.h"

#include <fmt/format.h>
#include <boost/asio.hpp>
#include "spdlog/spdlog.h"
#include <thread>

#undef SendMessage // MSVC crap

namespace net = boost::asio;

class ChatPrinter {
public:
    ChatPrinter(MessageQueue& mq) : mq_(mq) {
        mq_.Subscribe<MessagesReply>([this] (MessagesReply const& msg) { OnChatMessage(msg); });
    }

    void OnChatMessage(MessagesReply const& msg) {
        spdlog::get("chat_msg")->info("{}", msg.chat_msg);
    }


private:
    MessageQueue& mq_;
};

class Engine : public IEngine {
public:
    Engine(net::io_context& io_context) 
        : io_context_(io_context)
        , chat_printer_(mq_) 
    {
        mq_.SetSchedulerCallback([this] (auto callback) { net::post(io_context_, std::move(callback)); });
        
        mq_.Subscribe<NewConnection>([this] (NewConnection const& conn) {
            db_.AddCid({conn.cid});
            SendMessage(fmt::format("Well, hello {}. My name is {}", conn.cid, own_cid_), conn.cid);
        });

        mq_.Subscribe<MessageRequest> ([this] (MessageRequest const& req) {
            // TODO: more messages
            auto msgs = db_.GetLastMessages(req.cid, 1);
            if (msgs.empty()) {
                return;
            }
            server_->Deliver(msgs.back().text, req.from_cid);
        });

    }

    std::vector<std::string> GetKnownCid() override {
        return db_.GetKnownCids();
    }
    
    void SendMessage(std::string msg, std::string_view to_cid) override {
        // TODO: error handling
        db_.AddMessage(to_cid, {msg});
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

        server_ = std::make_unique<Server>(io_context_, mq_, own_cid_);

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
    }

private:
    std::shared_ptr<IDiscovery> discovery_service_;
    std::unique_ptr<Server> server_;

    net::io_context& io_context_;
    std::string own_cid_;
    std::string own_name_;
    MessageQueue mq_;
    ChatPrinter chat_printer_;
    Database db_;
};

std::shared_ptr<IEngine> CreateEngine(net::io_context& io_context) {
    return std::make_shared<Engine>(io_context);
}
