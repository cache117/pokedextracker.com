import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";

import { useSession } from "../../../hooks/contexts/use-session";
import type { Capture } from "../../../types";
import { API } from "../../../utils/api";
import { wrapper } from "../../library/layout";

import "../../../styles/availability.css";

interface Availability {
  dex_id: number;
  revision: number;
  hide_unavailable: boolean;
  unavailable_pokemon_ids: number[];
  default_unavailable_pokemon_ids: number[];
  is_owner: boolean;
}

export function useLeafGreenAvailability(dexId: number, enabled: boolean) {
  const { session } = useSession();
  const client = useQueryClient();
  const queryKey = ["LeafGreenAvailability", dexId, session?.username];
  const query = useQuery({
    queryKey,
    enabled,
    queryFn: () =>
      API.get<Availability>(`/personal/catalog/leaf-green/${dexId}`),
    retry: false,
  });
  const save = useMutation({
    mutationFn: (
      next: Pick<Availability, "hide_unavailable" | "unavailable_pokemon_ids">,
    ) =>
      API.post<Availability>(`/personal/catalog/leaf-green/${dexId}`, {
        revision: query.data?.revision,
        ...next,
      }),
    onSuccess: (data) => client.setQueryData(queryKey, data),
  });
  return { query, save };
}

interface Props {
  state: ReturnType<typeof useLeafGreenAvailability>;
  captures: Capture[];
}

export function LeafGreenAvailability({
  state: { query, save },
  captures,
}: Props) {
  const [draft, setDraft] = useState<number[] | null>(null);
  const [search, setSearch] = useState("");
  const [onlyUnavailable, setOnlyUnavailable] = useState(true);
  const [notice, setNotice] = useState("");
  const data = query.data;
  const ids = draft ?? data?.unavailable_pokemon_ids ?? [];
  const busy = save.isPending;
  const dirty = draft !== null;
  const persist = async (hide: boolean, unavailable: number[]) => {
    setNotice("");
    save.reset();
    try {
      await save.mutateAsync({
        hide_unavailable: hide,
        unavailable_pokemon_ids: unavailable,
      });
      setDraft(null);
      setNotice("Saved for this LeafGreen collection.");
    } catch {
      // The mutation's error is shown below; keep the draft for retrying.
    }
  };
  const options = captures.filter(
    ({ pokemon }) =>
      (!onlyUnavailable || ids.includes(pokemon.id)) &&
      `${pokemon.name} ${pokemon.national_id}`
        .toLowerCase()
        .includes(search.toLowerCase()),
  );
  return (
    <div className="availability-bar print:hidden">
      <div className={wrapper}>
        {!data ? (
          <p role={query.error ? "alert" : "status"}>
            {query.error
              ? "Couldn't load the unavailable list. All Pokémon remain visible."
              : "Loading LeafGreen availability…"}
            {query.error && (
              <button onClick={() => void query.refetch()}>Retry</button>
            )}
          </p>
        ) : (
          <>
            <label>
              <input
                checked={data.hide_unavailable}
                disabled={!data.is_owner || busy || dirty}
                id="hide-unavailable"
                name="hide-unavailable"
                onChange={(event) =>
                  void persist(
                    event.target.checked,
                    data.unavailable_pokemon_ids,
                  )
                }
                type="checkbox"
              />
              Hide Pokémon unobtainable in Switch FR/LG
            </label>
            <p className="availability-hint">
              {data.hide_unavailable
                ? `${data.unavailable_pokemon_ids.length} hidden`
                : `${data.unavailable_pokemon_ids.length} marked unavailable`}{" "}
              · This collection only. Evolution and local FR/LG trades count as
              obtainable.
            </p>
            <details>
              <summary>Edit unavailable list</summary>
              <p>
                The starting list comes from our Switch compatibility review
                (October 25, 2026). Uncheck a Pokémon when it becomes
                obtainable, or switch off the filter to see the full dex. Your
                edits survive reference imports.
              </p>
              <fieldset disabled={!data.is_owner || busy}>
                <label>
                  Find Pokémon
                  <input
                    onChange={(event) => setSearch(event.target.value)}
                    placeholder="Name or National Dex number"
                    value={search}
                  />
                </label>
                <label>
                  <input
                    checked={onlyUnavailable}
                    onChange={(event) =>
                      setOnlyUnavailable(event.target.checked)
                    }
                    type="checkbox"
                  />
                  Only show Pokémon marked unavailable
                </label>
                <div
                  aria-label="Unavailable Pokémon"
                  className="availability-list"
                >
                  {options.map(({ pokemon }) => (
                    <label key={pokemon.id}>
                      <input
                        checked={ids.includes(pokemon.id)}
                        onChange={(event) =>
                          setDraft(
                            event.target.checked
                              ? [...ids, pokemon.id].sort((a, b) => a - b)
                              : ids.filter((id) => id !== pokemon.id),
                          )
                        }
                        type="checkbox"
                      />
                      #{String(pokemon.national_id).padStart(3, "0")}{" "}
                      {pokemon.name}
                    </label>
                  ))}
                  {options.length === 0 && (
                    <p>
                      No matching Pokémon. Clear the search or turn off “Only
                      show Pokémon marked unavailable”.
                    </p>
                  )}
                </div>
                <div className="availability-actions">
                  <button
                    disabled={!dirty || busy}
                    onClick={() => void persist(data.hide_unavailable, ids)}
                  >
                    Save unavailable list
                  </button>
                  <button
                    disabled={!dirty || busy}
                    onClick={() => setDraft(null)}
                  >
                    Discard list changes
                  </button>
                  <button
                    onClick={() =>
                      setDraft([...data.default_unavailable_pokemon_ids])
                    }
                  >
                    Use reviewed starting list
                  </button>
                </div>
                {dirty && (
                  <p role="status">
                    List changes apply after you save. The boxes still use your
                    saved list.
                  </p>
                )}
              </fieldset>
            </details>
            {!data.is_owner && (
              <p className="availability-hint">
                Sign in as the owner to change this collection’s availability
                settings.
              </p>
            )}
          </>
        )}
        {save.error && (
          <p role="alert">
            {save.error.message}{" "}
            <button onClick={() => void query.refetch()}>
              Reload availability
            </button>
          </p>
        )}
        {notice && !dirty && <p role="status">{notice}</p>}
      </div>
    </div>
  );
}
