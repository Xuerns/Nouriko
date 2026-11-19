-- +goose Up
-- Seed 55 Indonesian student ("anak kos") food and ingredient reference prices.
-- Portable and idempotent across local PostgreSQL 18 and production Supabase PostgreSQL.

CREATE UNIQUE INDEX IF NOT EXISTS uq_food_prices_name_category
    ON food_prices (name, category);

INSERT INTO food_prices (name, price_idr, unit, category) VALUES
    -- Ready-to-eat meals (22 items)
    ('Nasi Putih', 4000, '1 porsi', 'meal'),
    ('Nasi Telor Dadar', 10000, '1 porsi', 'meal'),
    ('Nasi Telor Balado', 11000, '1 porsi', 'meal'),
    ('Nasi Tahu Tempe Orekan', 10000, '1 porsi', 'meal'),
    ('Nasi Sayur Sop & Tempe', 11000, '1 porsi', 'meal'),
    ('Nasi Sayur Asem & Tahu', 11000, '1 porsi', 'meal'),
    ('Nasi Pecel Sayur', 12000, '1 porsi', 'meal'),
    ('Gado-Gado Lontong', 14000, '1 porsi', 'meal'),
    ('Ketoprak', 13000, '1 porsi', 'meal'),
    ('Bubur Ayam', 12000, '1 porsi', 'meal'),
    ('Nasi Goreng Telur', 15000, '1 porsi', 'meal'),
    ('Nasi Ayam Geprek', 15000, '1 porsi', 'meal'),
    ('Nasi Ayam Goreng Kremes', 16000, '1 porsi', 'meal'),
    ('Nasi Ayam Bakar', 18000, '1 porsi', 'meal'),
    ('Nasi Pecel Lele', 15000, '1 porsi', 'meal'),
    ('Nasi Ikan Tongkol Balado', 14000, '1 porsi', 'meal'),
    ('Soto Ayam Nasi', 15000, '1 porsi', 'meal'),
    ('Nasi Capcay Sayur', 14000, '1 porsi', 'meal'),
    ('Nasi Rendang Warteg', 18000, '1 porsi', 'meal'),
    ('Nasi Uduk Telur', 12000, '1 porsi', 'meal'),
    ('Mie Goreng Telur Sayur', 13000, '1 porsi', 'meal'),
    ('Lotek Sayur', 12000, '1 porsi', 'meal'),

    -- Cooking ingredients: Protein (12 items)
    ('Telur Ayam', 2500, '1 butir', 'ingredient'),
    ('Tempe', 5000, '1 papan', 'ingredient'),
    ('Tahu Putih', 5000, '5 potong', 'ingredient'),
    ('Tahu Kuning', 6000, '5 potong', 'ingredient'),
    ('Dada Ayam Fillet', 12000, '250g', 'ingredient'),
    ('Paha Ayam', 10000, '250g', 'ingredient'),
    ('Ikan Tongkol', 10000, '250g', 'ingredient'),
    ('Ikan Kembung', 11000, '250g', 'ingredient'),
    ('Sarden Kaleng Kecil', 11000, '1 kaleng 155g', 'ingredient'),
    ('Sosis Ayam', 3000, '1 buah', 'ingredient'),
    ('Kacang Tanah', 6000, '100g', 'ingredient'),
    ('Susu UHT Plain', 6500, '250ml', 'ingredient'),

    -- Cooking ingredients: Carbohydrates (7 items)
    ('Beras', 15000, '1 kg', 'ingredient'),
    ('Oatmeal Instan', 12000, '200g', 'ingredient'),
    ('Roti Tawar Gandum', 14000, '1 bungkus', 'ingredient'),
    ('Kentang', 7000, '250g', 'ingredient'),
    ('Ubi Jalar', 5000, '250g', 'ingredient'),
    ('Mie Telur Kering', 5000, '1 bungkus 200g', 'ingredient'),
    ('Mie Instan', 3500, '1 bungkus', 'ingredient'),

    -- Cooking ingredients: Vegetables & Fruits (10 items)
    ('Bayam', 3000, '1 ikat', 'ingredient'),
    ('Kangkung', 3000, '1 ikat', 'ingredient'),
    ('Sawi Hijau', 3500, '1 ikat', 'ingredient'),
    ('Kol / Kubis', 4000, '1/2 buah', 'ingredient'),
    ('Wortel', 4000, '250g', 'ingredient'),
    ('Tomat', 4000, '250g', 'ingredient'),
    ('Tauge', 3000, '150g', 'ingredient'),
    ('Buncis', 4500, '200g', 'ingredient'),
    ('Pisang Cavendish', 4000, '1 buah', 'ingredient'),
    ('Pepaya Potong', 5000, '1 porsi', 'ingredient'),

    -- Cooking ingredients: Seasonings & Oils (4 items)
    ('Bawang Merah & Putih', 5000, '1 paket kecil', 'ingredient'),
    ('Cabai Rawit', 4000, '50g', 'ingredient'),
    ('Kecap Manis Sachet', 2000, '1 sachet', 'ingredient'),
    ('Minyak Goreng Pouch Kecil', 5000, '250ml', 'ingredient')
ON CONFLICT (name, category) DO UPDATE
SET
    price_idr = EXCLUDED.price_idr,
    unit = EXCLUDED.unit,
    updated_at = NOW();

-- +goose Down
DELETE FROM food_prices
WHERE (name, category) IN (
    ('Nasi Putih', 'meal'),
    ('Nasi Telor Dadar', 'meal'),
    ('Nasi Telor Balado', 'meal'),
    ('Nasi Tahu Tempe Orekan', 'meal'),
    ('Nasi Sayur Sop & Tempe', 'meal'),
    ('Nasi Sayur Asem & Tahu', 'meal'),
    ('Nasi Pecel Sayur', 'meal'),
    ('Gado-Gado Lontong', 'meal'),
    ('Ketoprak', 'meal'),
    ('Bubur Ayam', 'meal'),
    ('Nasi Goreng Telur', 'meal'),
    ('Nasi Ayam Geprek', 'meal'),
    ('Nasi Ayam Goreng Kremes', 'meal'),
    ('Nasi Ayam Bakar', 'meal'),
    ('Nasi Pecel Lele', 'meal'),
    ('Nasi Ikan Tongkol Balado', 'meal'),
    ('Soto Ayam Nasi', 'meal'),
    ('Nasi Capcay Sayur', 'meal'),
    ('Nasi Rendang Warteg', 'meal'),
    ('Nasi Uduk Telur', 'meal'),
    ('Mie Goreng Telur Sayur', 'meal'),
    ('Lotek Sayur', 'meal'),
    ('Telur Ayam', 'ingredient'),
    ('Tempe', 'ingredient'),
    ('Tahu Putih', 'ingredient'),
    ('Tahu Kuning', 'ingredient'),
    ('Dada Ayam Fillet', 'ingredient'),
    ('Paha Ayam', 'ingredient'),
    ('Ikan Tongkol', 'ingredient'),
    ('Ikan Kembung', 'ingredient'),
    ('Sarden Kaleng Kecil', 'ingredient'),
    ('Sosis Ayam', 'ingredient'),
    ('Kacang Tanah', 'ingredient'),
    ('Susu UHT Plain', 'ingredient'),
    ('Beras', 'ingredient'),
    ('Oatmeal Instan', 'ingredient'),
    ('Roti Tawar Gandum', 'ingredient'),
    ('Kentang', 'ingredient'),
    ('Ubi Jalar', 'ingredient'),
    ('Mie Telur Kering', 'ingredient'),
    ('Mie Instan', 'ingredient'),
    ('Bayam', 'ingredient'),
    ('Kangkung', 'ingredient'),
    ('Sawi Hijau', 'ingredient'),
    ('Kol / Kubis', 'ingredient'),
    ('Wortel', 'ingredient'),
    ('Tomat', 'ingredient'),
    ('Tauge', 'ingredient'),
    ('Buncis', 'ingredient'),
    ('Pisang Cavendish', 'ingredient'),
    ('Pepaya Potong', 'ingredient'),
    ('Bawang Merah & Putih', 'ingredient'),
    ('Cabai Rawit', 'ingredient'),
    ('Kecap Manis Sachet', 'ingredient'),
    ('Minyak Goreng Pouch Kecil', 'ingredient')
);

DROP INDEX IF EXISTS uq_food_prices_name_category;
