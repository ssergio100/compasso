ALTER TABLE device ADD COLUMN avatar_key TEXT NOT NULL DEFAULT 'cat'
  CHECK (avatar_key IN (
    'cat', 'dog', 'fox', 'rabbit', 'panda', 'owl', 'penguin', 'capybara',
    'cat_bow', 'rabbit_flower', 'panda_flower', 'fox_bow'
  ));

ALTER TABLE routine ADD COLUMN icon_key TEXT NOT NULL DEFAULT 'general'
  CHECK (icon_key IN (
    'study', 'reading', 'sleep', 'bath', 'meal', 'school', 'exercise',
    'chores', 'family', 'music', 'outdoor', 'general'
  ));

UPDATE routine SET icon_key = CASE
  WHEN lower(name) LIKE '%leit%' OR lower(name) LIKE '%livro%' THEN 'reading'
  WHEN lower(name) LIKE '%dorm%' OR lower(name) LIKE '%sono%' OR lower(name) LIKE '%descans%' THEN 'sleep'
  WHEN lower(name) LIKE '%banh%' OR lower(name) LIKE '%chuveir%' THEN 'bath'
  WHEN lower(name) LIKE '%almo%' OR lower(name) LIKE '%jantar%' OR lower(name) LIKE '%refei%' OR lower(name) LIKE '%lanche%' THEN 'meal'
  WHEN lower(name) LIKE '%escola%' OR lower(name) LIKE '%aula%' THEN 'school'
  WHEN lower(name) LIKE '%exerc%' OR lower(name) LIKE '%esport%' OR lower(name) LIKE '%futebol%' THEN 'exercise'
  WHEN lower(name) LIKE '%arrum%' OR lower(name) LIKE '%limp%' OR lower(name) LIKE '%organiz%' THEN 'chores'
  WHEN lower(name) LIKE '%famil%' THEN 'family'
  WHEN lower(name) LIKE '%music%' OR lower(name) LIKE '%músic%' OR lower(name) LIKE '%instrument%' THEN 'music'
  WHEN lower(name) LIKE '%passe%' OR lower(name) LIKE '%parque%' OR lower(name) LIKE '%brinc%' THEN 'outdoor'
  WHEN lower(name) LIKE '%estud%' OR lower(name) LIKE '%dever%' OR lower(name) LIKE '%licao%' OR lower(name) LIKE '%lição%' OR lower(name) LIKE '%tarefa%' THEN 'study'
  ELSE 'general'
END;

INSERT INTO schema_migrations(version) VALUES (11);
PRAGMA user_version = 11;
