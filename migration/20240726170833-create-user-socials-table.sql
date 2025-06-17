-- +migrate Up
CREATE TABLE `yoyojudge`.`user_socials` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `users_id` INT UNSIGNED NOT NULL,
  `social_type` ENUM('google') NOT NULL,
  `social_token` VARCHAR(255) NOT NULL,
  `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `iduser_socials_UNIQUE` (`id` ASC),
  UNIQUE INDEX `social_token_UNIQUE` (`social_token` ASC),
  INDEX `fk_user_socials_users_idx` (`users_id` ASC),
  UNIQUE INDEX `user_id_social_type_UNIQUE` (`users_id` ASC, `social_type` ASC),
  CONSTRAINT `fk_user_socials_users`
    FOREIGN KEY (`users_id`)
    REFERENCES `yoyojudge`.`users` (`id`)
    ON DELETE NO ACTION
    ON UPDATE NO ACTION)
ENGINE = InnoDB;
-- +migrate Down

