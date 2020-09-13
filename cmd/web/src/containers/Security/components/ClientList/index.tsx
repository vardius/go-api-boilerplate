import React, {useCallback, useEffect, useState} from "react";
import {Stack} from "@chakra-ui/core";
import {Client} from "src/types";
import {useApi, useUser} from "src/hooks";
import ClientCredentials from "../ClientCredentials";
import CreateClientDrawerButton from "./CreateClientDrawerButton";

const page = 1;
const limit = 999;

function ClientList() {
  const fetchJSON = useApi("auth");

  const [user] = useUser();
  const [isLoading, setIsLoading] = useState(true);
  const [clients, setClients] = useState<Array<Client>>([]);

  const fetchClients = useCallback(
    async ({page, limit}: { page: number; limit: number }) => {
      if (!user) {
        return null;
      }

      return await fetchJSON(`/clients`, "GET", new URLSearchParams({
        page: String(page),
        limit: String(limit),
      }));
    },
    [fetchJSON, user]
  );

  useEffect(() => {
    let mounted = true
    if (!user) {
      setIsLoading(false);
      return;
    }

    const load = async () => {
      try {
        const response = await fetchClients({page, limit});
        if (!mounted) {
          return
        }

        setIsLoading(false);
        setClients(response.clients || []);
      } catch (err) {
        if (!mounted) {
          return
        }

        setIsLoading(false);
      }
    };

    load();

    return function cleanup() {
      mounted = false
    }
  }, [user, fetchClients]);

  if (!user || isLoading) {
    return null;
  }

  const handleClientUpdate = async () => {
    const response = await fetchClients({page, limit});
    setClients(response.clients || []);
  };

  return (
    <Stack spacing={8}>
      <CreateClientDrawerButton onSuccess={handleClientUpdate}/>
      {clients.map(({id, secret, domain, redirect_url, scopes}) =>
        <ClientCredentials
          key={id}
          domain={domain}
          redirectURL={redirect_url}
          clientID={id}
          clientSecret={secret}
          scopes={scopes}
          onRemove={handleClientUpdate}
        />
      )}
    </Stack>
  );
}

export default ClientList;
