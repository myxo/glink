#pragma once

#include <any>
#include <atomic>
#include <functional>
#include <map>
#include <vector>
#include <typeindex>
#include <mutex>

#include <spdlog/spdlog.h>

class MessageQueue {
public:
    MessageQueue() = default;
    MessageQueue(MessageQueue const&) = delete;
    MessageQueue& operator=(MessageQueue const&) = delete;

    template<typename T, typename ...Args>
    void Send(Args... args) {
        std::lock_guard lock{mutex_};
        queue_.emplace_back(std::make_any<T>(args...));
        if (post_to_schedule_ && !in_scheduler_) {
            in_scheduler_ = true;
            // TODO: race condition on exit
            post_to_schedule_([this]{ ProcessAll(); });
        }
    }

    template<typename T>
    int Subscribe(std::function<void(T const&)> callback) {
        int token = token_count_.fetch_and(1);
        std::lock_guard lock{mutex_};
        subscribers_.emplace(
            std::type_index{ typeid(T) }, 
            SubInfo {
                token,
                [this, cb = std::move(callback)] (std::any const& any) {
                    cb(std::any_cast<T>(any));
                }
            }
        );
        return token;
    }

    void Unsubscribe(int token) {
        // TODO: 
    }

    void ProcessAll() {
        decltype(queue_) local_queue;
        {
            std::lock_guard lock{mutex_};
            in_scheduler_ = false;
            std::swap(queue_, local_queue);
        }
        spdlog::info("MEssageQueue, local_queue size = {}", local_queue.size());
        for (auto const& msg : local_queue) {
            std::lock_guard lock{mutex_}; // TODO: get rid off
            auto [it, end] = subscribers_.equal_range(msg.type());
            for (; it != end; ++it) {
                it->second.cb(msg);
            }
        }
    }

    void SetSchedulerCallback(std::function<void(std::function<void()>)> callback) {
        post_to_schedule_ = std::move(callback);
    }

private:
    struct SubInfo {
        int token;
        std::function<void(std::any const&)> cb;
    };

    std::recursive_mutex mutex_;
    std::vector<std::any> queue_;
    std::multimap<std::type_index, SubInfo> subscribers_;
    std::atomic_int token_count_{0};
    std::function<void(std::function<void()>)> post_to_schedule_;
    std::atomic_bool in_scheduler_{false};
};
