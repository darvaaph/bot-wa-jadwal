CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_identity
    ON classes(study_program, cohort_year, group_label);
