// Package cli provides the command-line interface for google-contacts.
package cli

import (
	"context"
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"google-contacts/internal/contacts"
)

// Version information
const Version = "0.1.0"

// RootCmd is the root command for the CLI.
var RootCmd = &cobra.Command{
	Use:   "google-contacts",
	Short: "Google Contacts Manager - Manage Google Contacts",
	Long:  "Create, search, and manage Google Contacts using Google People API v1",
}

// Create command flags
var (
	createFirstName string
	createLastName  string
	createPhone     string
	createEmail     string
	createCompany   string
	createPosition  string
	createNotes     string
)

// Command definitions
var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("google-contacts version %s\n", Version)
		},
	}

	createCmd = &cobra.Command{
		Use:   "create",
		Short: "Create a new contact",
		Long: `Create a new contact in Google Contacts.

Required fields:
  --firstname, -f: First name
  --lastname, -l:  Last name
  --phone, -p:     Phone number

Recommended fields:
  --company, -c:   Company name

Optional fields:
  --email, -e:     Email address
  --position, -r:  Role/position at company
  --notes, -n:     Notes about the contact`,
		Example: `  # Create contact with required fields only
  google-contacts create -f John -l Doe -p +33612345678

  # Create contact with all fields
  google-contacts create -f John -l Doe -p +33612345678 -c "Acme Inc" -r "CTO" -e john@acme.com -n "Met at conference"`,
		RunE: runCreate,
	}
)

func runCreate(cmd *cobra.Command, args []string) error {
	// Validate required fields
	if createFirstName == "" {
		return fmt.Errorf("first name is required (--firstname or -f)")
	}
	if createLastName == "" {
		return fmt.Errorf("last name is required (--lastname or -l)")
	}
	if createPhone == "" {
		return fmt.Errorf("phone number is required (--phone or -p)")
	}

	ctx := context.Background()

	// Get People API service
	srv, err := contacts.GetPeopleService(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize service: %w", err)
	}

	// Create the contact
	input := contacts.ContactInput{
		FirstName: createFirstName,
		LastName:  createLastName,
		Phone:     createPhone,
		Email:     createEmail,
		Company:   createCompany,
		Position:  createPosition,
		Notes:     createNotes,
	}

	created, err := srv.CreateContact(ctx, input)
	if err != nil {
		return err
	}

	// Display success message
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Println(green("Contact created successfully!"))
	fmt.Println()
	fmt.Printf("  %s: %s\n", cyan("Name"), created.DisplayName)
	fmt.Printf("  %s: %s\n", cyan("ID"), created.ResourceName)

	return nil
}

// Init initializes the CLI commands and flags.
func Init() {
	// Add version flag to root command
	RootCmd.Version = Version
	RootCmd.SetVersionTemplate("google-contacts version {{.Version}}\n")

	// Setup create command flags
	createCmd.Flags().StringVarP(&createFirstName, "firstname", "f", "", "First name (required)")
	createCmd.Flags().StringVarP(&createLastName, "lastname", "l", "", "Last name (required)")
	createCmd.Flags().StringVarP(&createPhone, "phone", "p", "", "Phone number (required)")
	createCmd.Flags().StringVarP(&createEmail, "email", "e", "", "Email address")
	createCmd.Flags().StringVarP(&createCompany, "company", "c", "", "Company name")
	createCmd.Flags().StringVarP(&createPosition, "position", "r", "", "Role/position at company")
	createCmd.Flags().StringVarP(&createNotes, "notes", "n", "", "Notes about the contact")

	// Register commands
	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(createCmd)
}
