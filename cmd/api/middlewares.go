package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")

				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) rateLimit(next http.Handler) http.Handler {
	//define a cliet struct to hold the rate limiter and the last seen of each client
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}

	// Declare a mutex and a map to hold the clients IP address and the Rate limiter
	var (
		mu sync.Mutex
		// clients is a map that holds the client structs
		clients = make(map[string]*client)
	)

	//lunch a background go routine that removes old entries from the map once every minute
	go func() {

		for {
			time.Sleep(time.Minute)
			//lock the mutex to prevent the rate limit checks from happenning while the cleanup is taking place
			mu.Lock()

			//loop through the map and check if the last seen time is older than 3 minutes
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}

			}

			//unlock themutex when the cleanup is done
			mu.Unlock()

		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract the Ip address from the request
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
		//lock the mutex to prevent the cod efrom being executed concurrently
		mu.Lock()

		//check if the ip address already exists in the map, if not found
		// append it to the map
		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst)}
		}

		//update the last seen time of the client
		clients[ip].lastSeen = time.Now()

		// Call the limiter.Allow() method to check if the client with current
		// IP is allowed to make that request. if the request is not allowed, unlock the mutex
		// and send a 429 Too Many Requests response to the client

		if !clients[ip].limiter.Allow() {

			mu.Unlock()
			app.rateLimitExceededResponse(w, r)

			return
		}
		// Very importantly, unlock the mutex before calling the next handler in the
		// chain. DON'T use defer to unlock the mutex, as that would mean
		// that the mutex isn't unlocked until all the handlers downstream of this
		// middleware have also returned.
		mu.Unlock()
		next.ServeHTTP(w, r)
	})
}