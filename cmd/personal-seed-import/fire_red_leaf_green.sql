-- Personal extension using the upstream game family, dex types and memberships.
-- IDs >= 1000000 are reserved for personal templates, away from upstream IDs.
-- Apply after every upstream import: its game family is deliberately unpublished.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM seed_dex_types WHERE id IN (1000001, 1000002)
    ) OR EXISTS (
        SELECT 1 FROM dex_types
        WHERE id IN (1000001, 1000002)
          AND (game_family_id <> 'fire_red_leaf_green'
               OR name <> CASE id WHEN 1000001 THEN 'Regional' ELSE 'Full National' END)
    ) THEN
        RAISE EXCEPTION 'Personal FireRed/LeafGreen dex IDs conflict with another template';
    END IF;

    -- Same base-species policy as upstream's X/Y and OR/AS Full National dexes.
    -- national_order excludes duplicate records used by later encounter families.
    IF (SELECT count(*) FROM pokemon p JOIN game_families gf ON gf.id=p.game_family_id
        WHERE p.national_id BETWEEN 1 AND 386 AND p.form IS NULL
          AND p.national_order >= 0 AND gf.generation <= 3) <> 386
       OR (SELECT count(DISTINCT p.national_id) FROM pokemon p
           JOIN game_families gf ON gf.id=p.game_family_id
           WHERE p.national_id BETWEEN 1 AND 386 AND p.form IS NULL
             AND p.national_order >= 0 AND gf.generation <= 3) <> 386 THEN
        RAISE EXCEPTION 'Expected exactly one original Pokemon per National Dex number 1-386';
    END IF;
END $$;

INSERT INTO dex_types (id, name, game_family_id, "order", tags, description)
VALUES
    (1000001, 'Regional', 'fire_red_leaf_green', 1, '{regional}',
     'Original Generation III Kanto Pokedex: 151 species, including Mew. Encounter locations have not been added yet.'),
    (1000002, 'Full National', 'fire_red_leaf_green', 2, '{"full national"}',
     'Original Generation III National Pokedex: 386 species, including Mythical Pokemon. Includes species requiring other games or events. Encounter locations have not been added yet.')
ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name, "order"=EXCLUDED."order",
    tags=EXCLUDED.tags, description=EXCLUDED.description;

-- Refuse to silently remove entries/captures if the upstream identity policy changes.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM dex_types_pokemon dtp JOIN pokemon p ON p.id=dtp.pokemon_id
        JOIN game_families gf ON gf.id=p.game_family_id
        WHERE dtp.dex_type_id IN (1000001, 1000002)
          AND NOT (p.national_id BETWEEN 1 AND CASE dtp.dex_type_id WHEN 1000001 THEN 151 ELSE 386 END
                   AND p.form IS NULL AND p.national_order >= 0 AND gf.generation <= 3)
    ) THEN
        RAISE EXCEPTION 'Existing personal FireRed/LeafGreen memberships need review';
    END IF;
END $$;

INSERT INTO dex_types_pokemon (dex_type_id, pokemon_id, box, "order", dex_number)
SELECT dt.id, p.id, NULL, p.national_id, p.national_id
FROM dex_types dt CROSS JOIN pokemon p JOIN game_families gf ON gf.id=p.game_family_id
WHERE dt.id IN (1000001, 1000002)
  AND p.national_id BETWEEN 1 AND CASE dt.id WHEN 1000001 THEN 151 ELSE 386 END
  AND p.form IS NULL AND p.national_order >= 0 AND gf.generation <= 3
ON CONFLICT (dex_type_id, pokemon_id) DO UPDATE SET
    "order"=EXCLUDED."order", dex_number=EXCLUDED.dex_number;

UPDATE game_families SET published=true WHERE id='fire_red_leaf_green';
