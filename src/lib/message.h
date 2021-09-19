#pragma once

#include <cstdio>
#include <cstdlib>
#include <cstring>


// TODO: different classes for buffer and parsed message?
// TODO: make single operation for reading message via asio
class Message {
public:
    constexpr const static int16_t kHeaderSize = 4;
    enum { max_body_length = 512 };

    Message() = default;

    Message(std::string_view payload) {
        payload_size_ = payload.size();
        assert(payload_size_ <= max_body_length);
        std::memcpy(body(), payload.data(), payload.size());
        encode_header();
    }

    const char* data() const { return data_; }

    char* data() { return data_; }

    std::size_t get_length() const { return kHeaderSize + payload_size_; }

    const char* body() const { return data_ + kHeaderSize; }

    char* body() { return data_ + kHeaderSize; }

    std::size_t body_length() const { return payload_size_; }

    bool decode_header() {
        char header[kHeaderSize + 1] = "";
        std::strncat(header, data_, kHeaderSize);
        payload_size_ = std::atoi(header);
        if (payload_size_ > max_body_length) {
            payload_size_ = 0;
            return false;
        }
        return true;
    }

 private:
    void encode_header() {
        char header[kHeaderSize + 1] = "";
        std::sprintf(header, "%4d", static_cast<int>(payload_size_));
        std::memcpy(data_, header, kHeaderSize);
    }

private:
    char data_[kHeaderSize + max_body_length] = {};
    std::size_t payload_size_{0};
};
