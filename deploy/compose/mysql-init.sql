-- One database and one user per service: neither can read the other's data.
-- Development credentials only; real deployments create these out of band.
CREATE DATABASE IF NOT EXISTS `auth` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
CREATE DATABASE IF NOT EXISTS `user` CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
CREATE USER IF NOT EXISTS 'auth'@'%' IDENTIFIED BY 'dev-only-auth';
CREATE USER IF NOT EXISTS 'user'@'%' IDENTIFIED BY 'dev-only-user';
GRANT ALL PRIVILEGES ON `auth`.* TO 'auth'@'%';
GRANT ALL PRIVILEGES ON `user`.* TO 'user'@'%';
