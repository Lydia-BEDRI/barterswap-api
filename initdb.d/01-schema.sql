-- BarterSwap Database Schema for MySQL

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Users table
CREATE TABLE IF NOT EXISTS users (
  id INT PRIMARY KEY AUTO_INCREMENT,
  pseudo VARCHAR(100) NOT NULL UNIQUE,
  bio TEXT,
  ville VARCHAR(100),
  credit_balance INT NOT NULL DEFAULT 10,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_pseudo (pseudo)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- User Skills table (M-to-1 relationship)
CREATE TABLE IF NOT EXISTS user_skills (
  id INT PRIMARY KEY AUTO_INCREMENT,
  user_id INT NOT NULL,
  nom VARCHAR(100) NOT NULL,
  niveau ENUM('débutant', 'intermédiaire', 'expert') NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_user_id (user_id),
  UNIQUE KEY unique_user_skill (user_id, nom)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Services table
CREATE TABLE IF NOT EXISTS services (
  id INT PRIMARY KEY AUTO_INCREMENT,
  provider_id INT NOT NULL,
  titre VARCHAR(255) NOT NULL,
  description TEXT,
  categorie ENUM('Informatique', 'Jardinage', 'Bricolage', 'Cuisine', 'Musique', 'Langues', 'Sport', 'Tutorat', 'Déménagement', 'Photographie', 'Animalier', 'Couture', 'Autre') NOT NULL,
  duree_minutes INT NOT NULL,
  credits INT NOT NULL,
  ville VARCHAR(100),
  actif BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (provider_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_provider_id (provider_id),
  INDEX idx_categorie (categorie),
  INDEX idx_ville (ville),
  INDEX idx_actif (actif),
  FULLTEXT INDEX idx_search (titre, description)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Exchanges table
CREATE TABLE IF NOT EXISTS exchanges (
  id INT PRIMARY KEY AUTO_INCREMENT,
  service_id INT NOT NULL,
  requester_id INT NOT NULL,
  owner_id INT NOT NULL,
  status ENUM('pending', 'accepted', 'rejected', 'cancelled', 'completed') DEFAULT 'pending',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE,
  FOREIGN KEY (requester_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_service_id (service_id),
  INDEX idx_requester_id (requester_id),
  INDEX idx_owner_id (owner_id),
  INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Credit Transactions table
CREATE TABLE IF NOT EXISTS credit_transactions (
  id INT PRIMARY KEY AUTO_INCREMENT,
  user_id INT NOT NULL,
  exchange_id INT NOT NULL,
  montant INT NOT NULL,
  type ENUM('earn', 'spend', 'refund') NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (exchange_id) REFERENCES exchanges(id) ON DELETE CASCADE,
  INDEX idx_user_id (user_id),
  INDEX idx_exchange_id (exchange_id),
  INDEX idx_type (type),
  UNIQUE KEY unique_exchange_credit_operation (exchange_id, user_id, type),
  CONSTRAINT chk_credit_transaction_amount CHECK (
    (type = 'spend' AND montant < 0)
    OR (type IN ('earn', 'refund') AND montant > 0)
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Reviews table
CREATE TABLE IF NOT EXISTS reviews (
  id INT PRIMARY KEY AUTO_INCREMENT,
  exchange_id INT NOT NULL,
  author_id INT NOT NULL,
  target_id INT NOT NULL,
  note INT NOT NULL CHECK (note >= 1 AND note <= 5),
  commentaire TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (exchange_id) REFERENCES exchanges(id) ON DELETE CASCADE,
  FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (target_id) REFERENCES users(id) ON DELETE CASCADE,
  INDEX idx_target_id (target_id),
  UNIQUE KEY unique_review_per_exchange (exchange_id, author_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
