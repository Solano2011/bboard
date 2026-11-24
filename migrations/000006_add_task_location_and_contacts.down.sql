ALTER TABLE tasks
    DROP COLUMN IF EXISTS district,
    DROP COLUMN IF EXISTS contact_phone,
    DROP COLUMN IF EXISTS work_address;
