package cmd

import (
	"crm-middleware/model"
	"crm-middleware/repo"
	"fmt"
	"os"
	"text/tabwriter"
)

func printClient(client *model.Client) {
	stderr("\n\tID:\t\t", client.ID)
	stderr("\tName:\t\t", client.Name)
	stderr("\tSecret:\t\t", client.Secret)
	stderr()
}

func clientList(args ...string) int {
	clients, err := repo.Clients.List()
	if err != nil {
		stderr("failed to list clients:", err)
		return 1
	}

	if len(clients) == 0 {
		stderr("no clients found")
		return 0
	}

	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tAPP\tID\tSECRET\tSTATUS\tUPDATED")
	for _, c := range clients {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			c.Name, orDash(c.App), c.ID, c.Secret, orDash(string(c.Status)),
			c.TimeUpdated.Format("2006-01-02 15:04:05"),
		)
	}
	w.Flush()
	return 0
}

func clientCreate(args ...string) int {
	var (
		name   string
		status = model.ClientStatusNone
	)

	for _, arg := range args {
		switch arg {
		case "--routing":
			status = model.ClientStatusRouting
		default:
			name = arg
		}
	}

	if name == "" {
		stderr("name cannot be empty")
		return 1
	}

	client, err := repo.Clients.Create(name, status)
	if err != nil {
		stderr("failed to create client:", err)
		return 1
	}

	printClient(client)
	return 0
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func clientReset(args ...string) int {
	if err := repo.Clients.ResetSecret(args[0]); err != nil {
		stderr("failed to reset client secret:", err)
		return 1
	}

	client, err := repo.Clients.Get(args[0])
	if err != nil {
		stderr("failed to retrieve updated client:", err)
		return 1
	}

	printClient(client)
	return 0
}
