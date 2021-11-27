#pragma once

#include "message_queue.h"

#pragma GCC diagnostic push
// implicit capture of ‘this’ via ‘[=]’ is deprecated in C++20
#pragma GCC diagnostic ignored "-Wdeprecated"

#include <cereal/archives/binary.hpp>
#include <cereal/types/memory.hpp>
#include <cereal/types/unordered_map.hpp>
#include <cereal/types/vector.hpp>
#include <cereal/archives/json.hpp>

#pragma GCC diagnostic pop

#include <cstdio>
#include <cstdlib>
#include <cstring>

enum class MsgType : uint16_t { 
    user_meta_request, 
    user_meta_reply, 
    new_messages, 
    messages_request, 
    messages_reply 
};

struct Header {
    MsgType type;
    uint16_t body_size;
};

// Struct for request meta after connection is established
struct UserMetaRequest {
    std::string from_cid;

    static constexpr MsgType type = MsgType::user_meta_request;

    template <class Archive>
    void serialize(Archive& ar) {
        ar(CEREAL_NVP(from_cid));
    }
};

struct UserMetaReply {
    std::string client_cid;
    std::string name;
    std::vector<std::string> rooms_cid;

    static constexpr MsgType type = MsgType::user_meta_reply;

    template <class Archive>
    void serialize(Archive& ar) {
        ar(CEREAL_NVP(client_cid), CEREAL_NVP(name), CEREAL_NVP(rooms_cid));
    }
};

struct MessageRequest {
    std::string from_cid;
    std::string cid;
    uint64_t from_index{0};
    // uuid last_msg_hash

    static constexpr MsgType type = MsgType::messages_request;

    template <class Archive>
    void serialize(Archive& ar) {
        ar(CEREAL_NVP(cid), CEREAL_NVP(from_index));
    }
};

struct MessagesReply {
    std::string chat_msg;

    static constexpr MsgType type = MsgType::messages_reply;

    template <class Archive>
    void serialize(Archive& ar) {
        ar(CEREAL_NVP(chat_msg));
    }
};

struct NewConnection {
    std::string cid;
    // ... endpoint
};

template<typename T>
auto DeserializePacket(std::string const& payload) {
    std::stringstream is(payload);
    T data_new;
    {
        cereal::JSONInputArchive archive_in(is);
        archive_in(data_new);
    }
    return data_new;
}

// TODO: different classes for buffer and parsed message?
// TODO: make single operation for reading message via asio
class Packet {
public:
    constexpr const static size_t kHeaderSize = 4;
    enum { max_body_length = 512 };

    Packet() = default;
    Packet(Packet &&) = default;
    Packet(Packet const&) = default;

    //Packet(std::string payload) {
    //    payload_ = payload;
    //    assert(payload_.size() <= max_body_length);
    //    std::memcpy(body(), payload.data(), payload.size());
    //    encode_header();
    //}

    template <typename T>
    Packet(T&& message) {
        {
            std::stringstream ss;
            {
                cereal::JSONOutputArchive archive(ss);
                archive(message);
            }
            payload_ = ss.str();
        }

        header_.type = T::type;
        header_.body_size = static_cast<uint16_t>(payload_.size());
        encode_header();
    }

    char* GetHeaderAsString() {
        return header_str_;
    }

    uint16_t GetBodySize() {
        return header_.body_size;
    }

    std::string const& GetPayload() const {
        return payload_;
    }

    std::string& GetPayloadMut() {
        return payload_;
    }

    bool decode_header() {
        std::memcpy(reinterpret_cast<void*>(&header_), header_str_, sizeof(Header));  // OMG
        payload_.reserve(header_.body_size);
        return true;
    }

    Header const& GetHeader() {
        return header_;
    }

private:
    void encode_header() {
        std::memcpy(header_str_, reinterpret_cast<void*>(&header_), sizeof(Header));  // OMG
    }

private:
    Header header_;
    char header_str_[kHeaderSize];
    std::string payload_;
};

// TODO: write payload to packet!?
inline void PutInMq(MessageQueue &mq, Header const& header, std::string const& payload) {
    try {
        switch (header.type) {
            case MsgType::user_meta_request:    mq.Send<UserMetaRequest>(DeserializePacket<UserMetaRequest>(payload)); break;
            case MsgType::user_meta_reply: mq.Send<UserMetaReply>(DeserializePacket<UserMetaReply>(payload)); break;
            case MsgType::new_messages: break;
            case MsgType::messages_request: mq.Send<MessageRequest>(DeserializePacket<MessageRequest>(payload)); break;
            case MsgType::messages_reply: mq.Send<MessagesReply>(DeserializePacket<MessagesReply>(payload)); break;
        }
    } catch (cereal::Exception const& e) { 
        // TODO: do not leak cereal
        // TODO: do not use warn?
        spdlog::warn("[PuInMq]: error while parse packet: {}\nPayload is {}", e.what(), payload);
    }

}

