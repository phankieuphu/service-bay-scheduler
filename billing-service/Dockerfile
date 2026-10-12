# ===== BUILD STAGE =====
# Same JDK the Gradle toolchain pins (build.gradle.kts). The Temurin images
# are multi-arch, so this builds natively on amd64 and arm64 alike.
FROM eclipse-temurin:25-jdk AS builder

WORKDIR /app

# Dependencies first, so a source-only change reuses this layer.
COPY gradlew settings.gradle.kts build.gradle.kts ./
COPY gradle gradle
RUN ./gradlew dependencies --no-daemon -q > /dev/null

COPY src src
RUN ./gradlew bootJar --no-daemon -q \
    && java -Djarmode=tools -jar build/libs/billing-service-*.jar extract --layers --launcher --destination extracted

# ===== RUNTIME STAGE =====
FROM eclipse-temurin:25-jre

RUN groupadd --system app && useradd --system --gid app --no-create-home app
WORKDIR /app

# Spring Boot's layers, least to most frequently changing, so a code change
# only replaces the last one.
COPY --from=builder /app/extracted/dependencies/ ./
COPY --from=builder /app/extracted/spring-boot-loader/ ./
COPY --from=builder /app/extracted/snapshot-dependencies/ ./
COPY --from=builder /app/extracted/application/ ./

USER app
EXPOSE 8084

# MaxRAMPercentage sizes the heap from the container's memory limit.
ENTRYPOINT ["java", "-XX:MaxRAMPercentage=75", "org.springframework.boot.loader.launch.JarLauncher"]
