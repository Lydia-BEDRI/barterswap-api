-- Test data for BarterSwap

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

INSERT INTO users (pseudo, bio, ville, credit_balance) VALUES
('alice', 'Développeuse passionnée', 'Paris', 15),
('bob', 'Bricoleur amateur', 'Lyon', 5),
('charlie', 'Professeur de musique', 'Marseille', 10),
('diane', 'Photographe', 'Toulouse', 10),
('eve', 'Jardinière enthousiaste', 'Bordeaux', 6);

INSERT INTO user_skills (user_id, nom, niveau) VALUES
(1, 'Python', 'expert'),
(1, 'JavaScript', 'intermédiaire'),
(2, 'Menuiserie', 'intermédiaire'),
(2, 'Électricité', 'débutant'),
(3, 'Piano', 'expert'),
(3, 'Guitare', 'intermédiaire'),
(4, 'Photographie', 'expert'),
(4, 'Retouche photo', 'expert'),
(5, 'Jardinage', 'expert'),
(5, 'Paysagisme', 'intermédiaire');

INSERT INTO services (provider_id, titre, description, categorie, duree_minutes, credits, ville, actif) VALUES
(1, 'Cours Python', 'Apprentissage Python de base', 'Tutorat', 120, 5, 'Paris', true),
(1, 'Aide JavaScript', 'Support pour projets web', 'Informatique', 90, 4, 'Paris', true),
(2, 'Réparation meuble', 'Remise en état mobilier', 'Bricolage', 180, 6, 'Lyon', true),
(2, 'Installation étagère', 'Pose d\'étagères murales', 'Bricolage', 60, 3, 'Lyon', true),
(3, 'Leçon piano', 'Cours particulier piano', 'Musique', 60, 5, 'Marseille', true),
(3, 'Leçon guitare', 'Cours guitare débutant', 'Musique', 45, 4, 'Marseille', true),
(4, 'Séance photo', 'Photos professionnelles', 'Photographie', 120, 8, 'Toulouse', true),
(4, 'Retouche photo', 'Édition d\'images', 'Photographie', 90, 5, 'Toulouse', true),
(5, 'Aménagement jardin', 'Design et création', 'Jardinage', 240, 10, 'Bordeaux', true),
(5, 'Taille arbres', 'Entretien végétal', 'Jardinage', 150, 6, 'Bordeaux', true);

INSERT INTO exchanges (service_id, requester_id, owner_id, status) VALUES
(1, 2, 1, 'completed'),
(3, 1, 2, 'pending'),
(6, 5, 3, 'accepted'),
(7, 2, 4, 'pending');

INSERT INTO credit_transactions (user_id, exchange_id, montant, type) VALUES
(2, 1, -5, 'spend'),
(1, 1, 5, 'earn'),
(5, 3, -4, 'spend');

INSERT INTO reviews (exchange_id, author_id, target_id, note, commentaire) VALUES
(1, 2, 1, 5, 'Excellent cours, très clair et pédagogue !'),
(1, 1, 2, 5, 'Client respectueux et attentif.');
