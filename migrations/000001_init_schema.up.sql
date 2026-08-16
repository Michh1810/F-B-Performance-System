CREATE TABLE menu_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    category VARCHAR(255) NOT NULL,
    current_price DECIMAL(10, 2) NOT NULL,
    cogs DECIMAL(10, 2) NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    menu_item_id UUID NOT NULL REFERENCES menu_items(id),
    quantity INTEGER NOT NULL,
    unit_price DECIMAL(10, 2) NOT NULL,
    sold_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    order_type VARCHAR(255)
);

CREATE TABLE yelp_reviews (
    review_id VARCHAR(22) PRIMARY KEY,
    source VARCHAR(22), 
    user_id VARCHAR(22),
    business_id VARCHAR(255),
    star SMALLINT NOT NULL CHECK (star >= 1 AND star <= 5),
    review_date DATE NOT NULL,
    review_text TEXT
);

CREATE TABLE google_reviews (
    review_id VARCHAR(22) PRIMARY KEY,
    source VARCHAR(22), 
    star SMALLINT NOT NULL CHECK (star >= 1 AND star <= 5),
    author_name VARCHAR(255),
    review_count INT NOT NULL,
    review_date DATE NOT NULL,
    review_text TEXT
);


CREATE OR REPLACE VIEW all_reviews AS
SELECT 
    review_id,
    source,
    star,
    review_text,
    review_date AS published_date,
    NULL AS yelp_user_id,
    NULL AS yelp_business_id,
    author_name AS google_author_name,
    review_count AS google_review_count
FROM google_reviews
UNION ALL
SELECT 
    review_id,
    source,
    star,
    review_text,
    review_date AS published_date,
    user_id AS yelp_user_id,
    business_id AS yelp_business_id,
    NULL AS google_author_name,
    NULL AS google_review_count
FROM yelp_reviews;




