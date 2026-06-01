-- seed_joke.sql
-- Run this script to insert a test joke of the day for the current date.
-- Ensure you have at least one user (admin) with id = 1.

INSERT INTO jokes (
    content, 
    sponsor_name, 
    sponsor_logo_url, 
    sponsor_website_url, 
    active_date, 
    created_by_id, 
    likes_count, 
    comments_count, 
    created_at, 
    updated_at
)
VALUES (
    'Why don''t scientists trust atoms? Because they make up everything!', 
    'Acme Corp', 
    'https://via.placeholder.com/150', 
    'https://example.com', 
    CURRENT_DATE, 
    1, 
    0, 
    0, 
    NOW(), 
    NOW()
)
ON CONFLICT (active_date) DO NOTHING;
