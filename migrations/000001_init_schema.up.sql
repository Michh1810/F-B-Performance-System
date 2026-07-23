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
    sold_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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


CREATE TABLE all_reviews (
    review_id VARCHAR(255) PRIMARY KEY, -- Use Yelp's ID or Google's ID
    source VARCHAR(22) NOT NULL,        -- Strictly 'google' or 'yelp'

    star SMALLINT NOT NULL CHECK (star >= 1 AND star <= 5),
    review_text TEXT,
    published_date DATE NOT NULL,
    
    yelp_user_id VARCHAR(22),
    yelp_business_id VARCHAR(255),
    
    -- Google-specific columns (Must allow NULLs)
    google_author_name VARCHAR(255),
    google_review_count INT,

    PRIMARY KEY (source, review_id)
);




