package codeownerizer

import (
	"context"
	"fmt"
	"strings"

	"log"

	"github.com/google/go-github/v69/github"
	"github.com/hmarr/codeowners"
)

const pushPermission = "push"

func AddUngrantedOwners(ctx context.Context, api *github.Client, org string, repo string, owners []codeowners.Owner) error {
	owners = uniqueOwners(owners)
	log.Printf("owners: %v", owners)

	teams, _, err := api.Repositories.ListTeams(ctx, org, repo, nil)
	if err != nil {
		return err
	}
	log.Println(teams)

	collaborators, _, err := api.Repositories.ListCollaborators(ctx, org, repo, nil)
	if err != nil {
		return err
	}
	log.Println(collaborators)

	for _, owner := range owners {
		switch owner.Type {
		case codeowners.TeamOwner:
			teamOwnerName := strings.Split(owner.String(), "/")[1]

			if !containsTeamOwner(teams, teamOwnerName) {
				log.Printf("team %s does not exist in the organization\n", teamOwnerName)
				continue
			}

			if !hasTeamOwnerSufficientPermission(teams, teamOwnerName) {
				resp, err := api.Teams.AddTeamRepoBySlug(ctx, org, teamOwnerName, org, repo, &github.TeamAddTeamRepoOptions{
					Permission: pushPermission,
				})
				log.Println(resp.Response.Header)
				if err != nil {
					log.Println(err.Error())
					continue
				}
				if err = github.CheckResponse(resp.Response); err != nil {
					log.Println(err.Error())
					continue
				}
				log.Printf("%s was added to the repo with the %s permission.\n", owner.String(), pushPermission)
			}
		case codeowners.UsernameOwner:
			userOwnerName := strings.TrimPrefix(owner.String(), "@")

			if !containsUserOwner(collaborators, userOwnerName) {
				log.Printf("user %s does not exist in the organization\n", userOwnerName)
				continue
			}

			if !hasUserOwnerSufficientPermission(collaborators, userOwnerName) {
				_, resp, err := api.Repositories.AddCollaborator(ctx, org, repo, userOwnerName, &github.RepositoryAddCollaboratorOptions{
					Permission: pushPermission,
				})
				log.Println(resp.Response.Header)
				if err != nil {
					log.Println(err.Error())
					continue
				}
				if err = github.CheckResponse(resp.Response); err != nil {
					log.Println(err.Error())
					continue
				}
				log.Printf("%s was added to the repo with the %s permission.\n", owner.String(), pushPermission)
			}
		case codeowners.EmailOwner:
			emailOwnerEmail := owner.String()
			userSearchResult, resp, err := api.Search.Users(ctx, fmt.Sprintf("%s in:email", emailOwnerEmail), nil)
			log.Println(resp.Response.Header)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			if err = github.CheckResponse(resp.Response); err != nil {
				log.Println(err.Error())
				continue
			}
			if len(userSearchResult.Users) > 1 {
				log.Printf("multiple users who has %s in email was found\n", emailOwnerEmail)
				continue
			}

			emailOwnerUsername := stringify(userSearchResult.Users[0].Login)

			if !containsUserOwner(collaborators, emailOwnerUsername) {
				log.Printf("user %s does not exist in the organization\n", emailOwnerUsername)
				continue
			}

			if !hasUserOwnerSufficientPermission(collaborators, emailOwnerUsername) {
				_, resp, err := api.Repositories.AddCollaborator(ctx, org, repo, emailOwnerUsername, &github.RepositoryAddCollaboratorOptions{
					Permission: pushPermission,
				})
				if err != nil {
					log.Println(err.Error())
					continue
				}
				if err = github.CheckResponse(resp.Response); err != nil {
					log.Println(err.Error())
					continue
				}
				log.Printf("%s was added to the repo with the %s permission.\n", owner.String(), pushPermission)
			}
		default:
			log.Printf("unknown owner type: %s\n", owner.Type)
			continue
		}
	}

	return nil
}
