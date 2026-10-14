# ===== BUILD STAGE =====
FROM eclipse-temurin:21-jdk AS builder

WORKDIR /app

# Dependencies first, so source-only changes reuse this layer.
COPY mvnw pom.xml ./
COPY .mvn .mvn
RUN ./mvnw -q -B dependency:go-offline

COPY src src
RUN ./mvnw -q -B package -DskipTests

# ===== RUNTIME STAGE =====
FROM eclipse-temurin:21-jre

WORKDIR /app

COPY --from=builder /app/target/*.jar /app/app.jar

EXPOSE 8085

CMD ["java", "-XX:MaxRAMPercentage=75", "-jar", "/app/app.jar"]
