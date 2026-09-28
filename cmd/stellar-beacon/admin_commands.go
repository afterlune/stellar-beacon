package main

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"os"
	"strings"
	"time"

	"github.com/afterlune/stellar-beacon/internal/domain/entity"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/config"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/migrations"
	"github.com/afterlune/stellar-beacon/internal/infrastructure/persistence/postgres/orm"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"xorm.io/xorm"
)

var bootstrapAdminEmail string

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "create or update the database schema",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		engine := ormInit.GetEngine()
		if engine == nil {
			return errors.New("database engine could not be initialized")
		}
		defer engine.Close()
		ctx, cancel := context.WithTimeout(cmd.Context(), 2*time.Minute)
		defer cancel()
		if err := migrations.Apply(ctx, engine); err != nil {
			return err
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), "database migrations are up to date")
		return err
	},
}

var bootstrapAdminCmd = &cobra.Command{
	Use:   "bootstrap-admin",
	Short: "create the first administrator account",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Validate(); err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}
		if err := validateAdminEmail(bootstrapAdminEmail); err != nil {
			return err
		}
		password, err := readAdminPassword()
		if err != nil {
			return err
		}
		engine := ormInit.GetEngine()
		if engine == nil {
			return errors.New("database engine could not be initialized")
		}
		defer engine.Close()
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()
		if err := createFirstAdmin(ctx, engine, bootstrapAdminEmail, password); err != nil {
			return err
		}
		_, err = fmt.Fprintf(cmd.OutOrStdout(), "administrator account created for %s\n", bootstrapAdminEmail)
		return err
	},
}

func init() {
	bootstrapAdminCmd.Flags().StringVar(&bootstrapAdminEmail, "email", "", "administrator email address")
	_ = bootstrapAdminCmd.MarkFlagRequired("email")
	rootCmd.AddCommand(migrateCmd, bootstrapAdminCmd)
}

func validateAdminEmail(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) == 0 || len(email) > 50 {
		return errors.New("email must be between 1 and 50 bytes")
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || parsed.Address != email {
		return errors.New("email must be a valid address without a display name")
	}
	bootstrapAdminEmail = email
	return nil
}

func readAdminPassword() ([]byte, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errors.New("bootstrap-admin requires an interactive terminal to read the password securely")
	}
	if _, err := fmt.Fprint(os.Stderr, "Administrator password (12-72 bytes): "); err != nil {
		return nil, err
	}
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	_, _ = fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("read administrator password: %w", err)
	}
	if len(password) < 12 || len(password) > 72 {
		return nil, errors.New("password must be between 12 and 72 bytes")
	}
	if _, err := fmt.Fprint(os.Stderr, "Confirm password: "); err != nil {
		return nil, err
	}
	confirmation, err := term.ReadPassword(int(os.Stdin.Fd()))
	_, _ = fmt.Fprintln(os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("read password confirmation: %w", err)
	}
	if string(password) != string(confirmation) {
		return nil, errors.New("password confirmation does not match")
	}
	return password, nil
}

func createFirstAdmin(ctx context.Context, engine *xorm.Engine, email string, password []byte) error {
	passwordHash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash administrator password: %w", err)
	}
	session := engine.NewSession().Context(ctx)
	defer session.Close()
	if err := session.Begin(); err != nil {
		return fmt.Errorf("begin administrator setup: %w", err)
	}
	defer session.Rollback()

	var adminCount int64
	if _, err := session.SQL(`
		SELECT COUNT(1)
		FROM t_user_role ur
		JOIN t_role r ON r.id = ur.role_id
		WHERE r.role_name = 'admin'
	`).Get(&adminCount); err != nil {
		return fmt.Errorf("check existing administrator: %w", err)
	}
	if adminCount > 0 {
		return errors.New("an administrator account already exists; refusing to create or reset another account")
	}
	var role entity.TRole
	found, err := session.Where("role_name = ? AND is_disable = 0", "admin").Get(&role)
	if err != nil {
		return fmt.Errorf("find administrator role: %w", err)
	}
	if !found {
		return errors.New("administrator role is missing; run the database migrations first")
	}
	var accountCount int64
	if _, err := session.SQL("SELECT COUNT(1) FROM t_user_auth WHERE lower(username) = ?", email).Get(&accountCount); err != nil {
		return fmt.Errorf("check existing account: %w", err)
	}
	if accountCount > 0 {
		return errors.New("an account with this email already exists")
	}

	nickname := strings.SplitN(email, "@", 2)[0]
	info := entity.TUserInfo{
		Email: email, Nickname: nickname, Avatar: "", ProfileLinksJSON: "[]", IsSubscribe: 0, IsDisable: 0,
	}
	if _, err := session.Insert(&info); err != nil {
		return fmt.Errorf("create administrator profile: %w", err)
	}
	auth := entity.TUserAuth{
		UserInfoId: info.Id, Username: email, Password: string(passwordHash), LoginType: 1,
		IpAddress: "127.0.0.1", IpSource: "bootstrap",
	}
	if _, err := session.Insert(&auth); err != nil {
		return fmt.Errorf("create administrator credentials: %w", err)
	}
	if _, err := session.Insert(&entity.TUserRole{UserId: info.Id, RoleId: role.Id}); err != nil {
		return fmt.Errorf("assign administrator role: %w", err)
	}
	if err := session.Commit(); err != nil {
		return fmt.Errorf("commit administrator setup: %w", err)
	}
	return nil
}
