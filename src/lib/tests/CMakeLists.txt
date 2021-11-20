add_executable(core_test)

target_sources(core_test PRIVATE
        mq_test.cpp
)

target_include_directories(core_test PUBLIC ../"include")

target_link_libraries(core_test PUBLIC chatik_core ${CONAN_LIBS})
