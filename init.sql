DROP TABLE IF EXISTS boms;

CREATE TABLE boms (
    id SERIAL PRIMARY KEY,
    bom_code VARCHAR(50) NOT NULL,
    part_reference VARCHAR(50) NOT NULL,
    part_name VARCHAR(100) NOT NULL,
    part_description TEXT,
    quantity INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bom_code ON boms (bom_code);

INSERT INTO boms (bom_code, part_reference, part_name, part_description, quantity) VALUES
('BOM-001', 'PR-1001', 'Base Plate', 'Main structural base', 1),
('BOM-001', 'PR-1002', 'Top Plate', 'Top cover plate', 1),
('BOM-001', 'PR-1003', 'Mounting Component', 'Set of 4 mounting brackets', 4),
('BOM-002', 'PR-2011', 'Drawer Stopper', 'Prevents drawer from falling', 2),
('BOM-002', 'PR-2012', 'Handle Drawer', 'Standard pull handle', 1),
('BOM-002', 'PR-2013', 'Roda Drawer', 'Caster wheel for mobility', 4),
('BOM-003', 'PR-3001', 'Connection Power Supply', 'Main power inlet and switch', 1),
('BOM-003', 'PR-3002', 'Locking Mechanism', 'Keyed lock for security', 1);

\echo '✅ Database initialized and seeded with BOM data.'

DROP TABLE IF EXISTS actionable_items;
DROP TYPE IF EXISTS item_type_enum;
DROP TYPE IF EXISTS item_status_enum;

CREATE TYPE item_type_enum AS ENUM ('SHORTAGE', 'UNLISTED');
CREATE TYPE item_status_enum AS ENUM ('BARU_MASUK', 'DITINDAKLANJUTI', 'SELESAI');

CREATE TABLE actionable_items (
    id SERIAL PRIMARY KEY,
    bom_code VARCHAR(50) NOT NULL,
    part_name VARCHAR(100) NOT NULL,
    item_type item_type_enum NOT NULL,
    quantity_diff INTEGER NOT NULL, 
    status item_status_enum NOT NULL DEFAULT 'BARU_MASUK',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
   NEW.updated_at = NOW();
   RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_actionable_items_updated_at
BEFORE UPDATE ON actionable_items
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

\echo '✅ Tabel actionable_items dibuat.'

DROP TABLE IF EXISTS detection_results;


CREATE TABLE detection_results (
    id SERIAL PRIMARY KEY,
    bom_code VARCHAR(50) NOT NULL UNIQUE,
    original_image TEXT NOT NULL,
    annotated_image TEXT NOT NULL,
    comparison_result_json JSONB NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TRIGGER update_detection_results_updated_at
BEFORE UPDATE ON detection_results
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

\echo '✅ Tabel detection_results diperbarui dengan kolom original_image.'