#include "../message_queue.h"

#define CATCH_CONFIG_MAIN 
#include <catch2/catch.hpp>

class TestMsg {};
class TestMsg2 {};

TEST_CASE( "Message queue", "[send_to_one]" ) {
    int count = 0;
    MessageQueue mq;

    mq.Subscribe<TestMsg>([&count] (TestMsg const&) mutable { count++; });
    mq.Send<TestMsg>();
    mq.ProcessAll();

    REQUIRE(count == 1);
}

TEST_CASE( "send many types", "[]" ) {
    int count = 0;
    MessageQueue mq;

    mq.Subscribe<TestMsg>([&count] (TestMsg const&) mutable { count++; });
    mq.Send<TestMsg>();
    mq.Send<TestMsg2>();

    mq.ProcessAll();

    REQUIRE(count == 1);
}

TEST_CASE( "send many consumer", "[]" ) {
    int count1 = 0;
    int count2 = 0;
    MessageQueue mq;

    mq.Subscribe<TestMsg>([&count1] (TestMsg const&) mutable { count1++; });
    mq.Subscribe<TestMsg2>([&count1] (TestMsg2 const&) mutable { count1++; });

    mq.Subscribe<TestMsg>([&count2] (TestMsg const&) mutable { count2++; });

    mq.Send<TestMsg>();
    mq.Send<TestMsg2>();

    mq.ProcessAll();

    REQUIRE(count1 == 2);
    REQUIRE(count2 == 1);
}

TEST_CASE( "scheduler callback", "[]" ) {
    int count = 0;
    MessageQueue mq;

    mq.SetSchedulerCallback([&] (std::function<void()> post) { count++; });

    mq.Send<TestMsg>();
    REQUIRE(count == 1);

    mq.Send<TestMsg2>();
    REQUIRE(count == 1);

    mq.ProcessAll();

    mq.Send<TestMsg2>();
    REQUIRE(count == 2);
}
