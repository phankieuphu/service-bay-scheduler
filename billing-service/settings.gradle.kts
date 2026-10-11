plugins {
    // Lets the toolchain below download the pinned JDK when it isn't installed.
    id("org.gradle.toolchains.foojay-resolver-convention") version "1.0.0"
}

rootProject.name = "billing-service"
