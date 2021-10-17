#pragma once

//#include <boost/pfr.hpp>
#include <cereal/archives/binary.hpp>
#include <cereal/types/memory.hpp>
#include <cereal/types/unordered_map.hpp>
#include <cereal/archives/json.hpp>

#include <cstdio>
#include <cstdlib>
#include <cstring>

enum class MsgType : uint16_t { user_meta_request, user_meta_reply, new_messages, messages_request, messages_reply };

struct Header {
    MsgType type;
    uint16_t body_size;
};

struct MessagesReply {
    std::string chat_msg;

    static constexpr MsgType type = MsgType::messages_reply;

    template <class Archive>
    void serialize(Archive& ar) {
        ar(CEREAL_NVP(chat_msg));
    }
};

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

private:
    void encode_header() {
        std::memcpy(header_str_, reinterpret_cast<void*>(&header_), sizeof(Header));  // OMG
    }

private:
    Header header_;
    char header_str_[kHeaderSize];
    std::string payload_;
};
