import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { Nav } from '../library/Nav';
import { API } from '../../utils/api';
import { useSession } from '../../hooks/contexts/use-session';
import { queryClient } from '../../utils/query-client';
import '../../styles/manage.scss';

interface SourceGame {
 id: string; name: string; generation: number; platform: string; groups: string[]; route_id: string; included: boolean; owned: boolean;
}
interface TransferRoute {
 id: string; name: string; reaches_home: boolean; enabled: boolean; one_way: boolean; starts_at: string | null; ends_at: string | null;
}
interface EncounterOverride {
 pokemon_id: number; game_id: string; values: string[];
}
interface Catalogue {
 revision: number; is_owner?: boolean; settings: { version: number; home_mode: string; hidden_groups: string[]; games: SourceGame[]; routes: TransferRoute[] }; overrides: EncounterOverride[];
}
interface PokemonOption {
 id: number; name: string; national_id: number; form: string | null;
}
interface Encounter {
 game_id: string; values: string[];
}

const splitLines = (value: string) => value.split('\n').map((line) => line.trim()).filter(Boolean);
const localDate = (value: string | null) => {
  if (!value) return ''; const date = new Date(value); return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16);
};
const payload = (value: Catalogue) => ({ revision: value.revision, settings: value.settings, overrides: value.overrides });
function validConfiguration (value: Catalogue) {
  try {
    return value.settings.version === 1 && ['all', 'current', 'owned'].includes(value.settings.home_mode)
      && Array.isArray(value.settings.hidden_groups) && value.settings.hidden_groups.every((group) => typeof group === 'string')
      && Array.isArray(value.settings.games) && value.settings.games.length <= 1000 && value.settings.games.every((game) => typeof game.id === 'string' && typeof game.name === 'string' && typeof game.platform === 'string' && Number.isInteger(game.generation) && typeof game.route_id === 'string' && typeof game.included === 'boolean' && typeof game.owned === 'boolean' && Array.isArray(game.groups) && game.groups.every((group) => typeof group === 'string'))
      && Array.isArray(value.settings.routes) && value.settings.routes.length <= 100 && value.settings.routes.every((route) => typeof route.id === 'string' && typeof route.name === 'string' && typeof route.reaches_home === 'boolean' && typeof route.enabled === 'boolean' && typeof route.one_way === 'boolean' && [route.starts_at, route.ends_at].every((date) => date === null || (typeof date === 'string' && !Number.isNaN(Date.parse(date)))))
      && Array.isArray(value.overrides) && value.overrides.length <= 50000 && value.overrides.every((row) => Number.isInteger(row.pokemon_id) && typeof row.game_id === 'string' && Array.isArray(row.values) && row.values.every((line) => typeof line === 'string'));
  } catch (_) {
    return false;
  }
}

export function Manage () {
  const { session } = useSession();
  const [data, setData] = useState<Catalogue | null>(null);
  const [saved, setSaved] = useState('');
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [busy, setBusy] = useState(false);
  const [search, setSearch] = useState('');
  const [pokemon, setPokemon] = useState<PokemonOption[]>([]);
  const [pokemonSearch, setPokemonSearch] = useState('');
  const [selectedPokemon, setSelectedPokemon] = useState(1);
  const [selectedGame, setSelectedGame] = useState('');
  const [baseline, setBaseline] = useState<Encounter[]>([]);
  const [encounterText, setEncounterText] = useState('');
  const [encounterLoading, setEncounterLoading] = useState(false);
  const [pendingImport, setPendingImport] = useState<Catalogue | null>(null);
  const [newGame, setNewGame] = useState({ id: '', name: '', platform: 'Switch', generation: 9, route_id: 'home' });

  const load = async () => {
    setError('');
    try {
      const [catalogue, options] = await Promise.all([API.get<Catalogue>('/personal/catalog'), API.get<PokemonOption[]>('/personal/catalog/pokemon')]);
      setData(catalogue); setSaved(JSON.stringify(payload(catalogue))); setPokemon(options);
      setSelectedGame((old) => old || catalogue.settings.games.find((game) => game.id !== 'home')?.id || '');
    } catch (err) {
      setError((err as Error).message);
    }
  };
  useEffect(() => {
    document.title = 'Manage games and sources | Personal Pokédex'; void load();
  }, [session?.username]);
  useEffect(() => {
    let current = true; setEncounterLoading(true);
    API.get<Encounter[]>(`/personal/catalog/encounters/${selectedPokemon}`).then((rows) => {
      if (current) setBaseline(rows);
    }).catch((err: Error) => {
      if (current) setError(err.message);
    }).finally(() => {
      if (current) setEncounterLoading(false);
    });
    return () => {
      current = false;
    };
  }, [selectedPokemon]);
  const selectedOverride = data?.overrides.find((row) => row.pokemon_id === selectedPokemon && row.game_id === selectedGame);
  const importedLines = baseline.find((row) => row.game_id === selectedGame)?.values || [];
  const effectiveText = (selectedOverride?.values || importedLines).join('\n');
  useEffect(() => {
    setEncounterText(effectiveText);
  }, [effectiveText, selectedGame, selectedPokemon]);

  const groups = useMemo(() => [...new Set(data?.settings.games.flatMap((game) => game.groups).filter(Boolean) || [])].sort(), [data]);
  if (!data) return <><Nav /><main className="catalogue"><h1>Manage games and sources</h1><p role="alert">{error || 'Loading your catalogue…'}</p></main></>;
  const editable = !!data.is_owner;
  const dirty = JSON.stringify(payload(data)) !== saved;
  const selectedMon = pokemon.find((mon) => mon.id === selectedPokemon);
  const patchGame = (id: string, patch: Partial<SourceGame>) => setData({ ...data, settings: { ...data.settings, games: data.settings.games.map((game) => game.id === id ? { ...game, ...patch } : game) } });
  const patchRoute = (id: string, patch: Partial<TransferRoute>) => setData({ ...data, settings: { ...data.settings, routes: data.settings.routes.map((route) => route.id === id ? { ...route, ...patch } : route) } });
  const save = async () => {
    setBusy(true); setError(''); setNotice('');
    try {
      const result = await API.post<Catalogue>('/personal/catalog', payload(data)); const next = { ...result, is_owner: editable }; setData(next); setSaved(JSON.stringify(payload(next))); await queryClient.invalidateQueries(); setNotice('Saved. Your source rules and encounter edits are now active.');
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const keepEncounter = () => {
    const row = { pokemon_id: selectedPokemon, game_id: selectedGame, values: splitLines(encounterText) };
    setData({ ...data, overrides: [...data.overrides.filter((item) => !(item.pokemon_id === selectedPokemon && item.game_id === selectedGame)), row] });
    setNotice('Encounter override added to your changes. Save changes to apply it.');
  };
  const restoreEncounter = () => {
    setData({ ...data, overrides: data.overrides.filter((item) => !(item.pokemon_id === selectedPokemon && item.game_id === selectedGame)) }); setEncounterText(importedLines.join('\n')); setNotice('Imported encounter text restored in your changes. Save changes to apply it.');
  };
  const exportConfiguration = () => {
    const blob = new Blob([JSON.stringify(payload(data), null, 2)], { type: 'application/json' }); const url = URL.createObjectURL(blob); const link = document.createElement('a'); link.href = url; link.download = 'personal-pokedex-catalogue.json'; link.click(); URL.revokeObjectURL(url);
  };
  const readImport = async (file?: File) => {
    if (!file) return; setError('');
    try {
      if (file.size > 8 * 1024 * 1024) throw new Error('Choose a configuration smaller than 8 MB.'); const parsed = JSON.parse(await file.text()) as Catalogue; if (!validConfiguration(parsed)) throw new Error('This is not a supported catalogue export.'); setPendingImport(parsed);
    } catch (err) {
      setError((err as Error).message);
    }
  };
  const addGame = () => {
    const gameId = `game_${newGame.name.normalize('NFKD').replace(/[\u0300-\u036f]/g, '').toLowerCase().replace(/[^a-z0-9]+/g, '_').replace(/_$/, '')}`.slice(0, 60);
    if (!newGame.name.trim() || data.settings.games.some((game) => game.id === gameId || game.name.toLowerCase() === newGame.name.trim().toLowerCase())) {
      setError('Give this source game a name that distinguishes it from existing editions.'); return;
    }
    setData({ ...data, settings: { ...data.settings, games: [...data.settings.games, { ...newGame, id: gameId, name: newGame.name.trim(), groups: [newGame.platform], included: true, owned: true }] } });
    setSelectedGame(gameId); setNewGame({ ...newGame, id: '', name: '' }); setNotice('Source game added to your changes. Save to make it available.'); setError('');
  };
  const addRoute = () => {
    let suffix = data.settings.routes.length + 1;
    while (data.settings.routes.some((route) => route.id === `route_${suffix}`)) suffix++;
    setData({ ...data, settings: { ...data.settings, routes: [...data.settings.routes, { id: `route_${suffix}`, name: 'New transfer route', reaches_home: false, enabled: true, one_way: false, starts_at: null, ends_at: null }] } });
  };

  return <><Nav /><main className="catalogue">
    <header className="catalogue-heading"><div><p className="eyebrow">PERSONAL POKÉDEX</p><h1>Manage games and sources</h1><p>Keep your catalogue useful as games and services change.</p></div><Link to={`/u/${session?.username || 'cache117'}`}>Back to my dexes →</Link></header>
    {!editable && <p className="catalogue-notice">You can browse the catalogue. <Link to="/login">Sign in as the owner</Link> to edit it.</p>}
    <div className="catalogue-save"><span>{dirty ? 'You have unsaved changes' : 'All changes saved'} · {data.settings.games.length} games · {data.overrides.length} encounter overrides</span><button disabled={!editable || busy || !dirty} onClick={() => void save()}>{busy ? 'Saving…' : 'Save changes'}</button></div>
    {error && <p className="catalogue-error" role="alert">{error}</p>}{notice && <p className="catalogue-notice" role="status">{notice}</p>}

    <section className="catalogue-card"><h2>Sources for HOME collections</h2><p>These rules filter encounter suggestions in HOME dexes. Your Pokémon checklist and caught status stay the same. Other game dexes keep their own encounter information.</p>
      <fieldset disabled={!editable || busy}><label>Show<select onChange={(e) => setData({ ...data, settings: { ...data.settings, home_mode: e.target.value } })} value={data.settings.home_mode}><option value="all">All included sources, including historical routes</option><option value="current">Sources that can currently reach HOME</option><option value="owned">Only my games that can currently reach HOME</option></select></label>
        <p>Hide groups:</p><div className="catalogue-chips">{groups.map((group) => <label key={group}><input checked={data.settings.hidden_groups.includes(group)} onChange={(e) => setData({ ...data, settings: { ...data.settings, hidden_groups: e.target.checked ? [...data.settings.hidden_groups, group] : data.settings.hidden_groups.filter((value) => value !== group) } })} type="checkbox" />{group}</label>)}</div></fieldset>
      <p className="catalogue-hint">“Include” and hidden groups apply in every HOME view. Current-source views also check the route, its dates, and whether you own the game. Unknown routes are excluded from current sources.</p>
    </section>

    <section className="catalogue-card"><div className="catalogue-section-heading"><div><h2>Games and editions</h2><p>Platform, groups, and transfer route are separate. Give re-releases their own source entry.</p></div><input aria-label="Filter source games" onChange={(e) => setSearch(e.target.value)} placeholder="Find a game, platform, or group" value={search} /></div>
      <fieldset disabled={!editable || busy}><div className="catalogue-table"><table><thead><tr><th>Game</th><th>Platform</th><th>Groups, separated by commas</th><th>Transfer route</th><th>Include</th><th>I own it</th></tr></thead><tbody>{data.settings.games.filter((game) => `${game.name} ${game.platform} ${game.groups.join(' ')}`.toLowerCase().includes(search.toLowerCase())).map((game) => <tr key={game.id}>
        <td><input aria-label={`Name for ${game.id}`} onChange={(e) => patchGame(game.id, { name: e.target.value })} value={game.name} /></td>
        <td><input aria-label={`Platform for ${game.name}`} onChange={(e) => patchGame(game.id, { platform: e.target.value })} value={game.platform} /></td>
        <td><input aria-label={`Groups for ${game.name}`} onChange={(e) => patchGame(game.id, { groups: e.target.value ? e.target.value.split(',').map((value) => value.trim()) : [] })} value={game.groups.join(', ')} /></td>
        <td><select aria-label={`Transfer route for ${game.name}`} onChange={(e) => patchGame(game.id, { route_id: e.target.value })} value={game.route_id}>{data.settings.routes.map((route) => <option key={route.id} value={route.id}>{route.name}</option>)}</select></td>
        <td><input aria-label={`Include ${game.name}`} checked={game.included} onChange={(e) => patchGame(game.id, { included: e.target.checked })} type="checkbox" /></td>
        <td><input aria-label={`I own ${game.name}`} checked={game.owned} onChange={(e) => patchGame(game.id, { owned: e.target.checked })} type="checkbox" /></td>
      </tr>)}</tbody></table></div></fieldset>
      <details><summary>Add a source game</summary><p>Add a game to encounter suggestions. Creating its playable dex list is a separate step.</p><fieldset disabled={!editable || busy}><div className="catalogue-grid">
        <label>Name<input onChange={(e) => setNewGame({ ...newGame, name: e.target.value })} placeholder="FireRed (Switch)" value={newGame.name} /></label>
        <label>Platform<input onChange={(e) => setNewGame({ ...newGame, platform: e.target.value })} value={newGame.platform} /></label>
        <label>Pokémon generation<input max="100" min="1" onChange={(e) => setNewGame({ ...newGame, generation: Number(e.target.value) })} type="number" value={newGame.generation} /></label>
        <label>Transfer route<select onChange={(e) => setNewGame({ ...newGame, route_id: e.target.value })} value={newGame.route_id}>{data.settings.routes.map((route) => <option key={route.id} value={route.id}>{route.name}</option>)}</select></label>
      </div><button className="secondary" onClick={addGame}>Add to catalogue</button></fieldset></details>
    </section>

    <section className="catalogue-card"><h2>Transfer routes and availability</h2><p>Update a shared route once to affect every game that uses it. Dates use your browser’s local time. End dates are exclusive.</p><fieldset disabled={!editable || busy}><div className="catalogue-routes">{data.settings.routes.map((route) => <article key={route.id}>
      <label>Route name<input onChange={(e) => patchRoute(route.id, { name: e.target.value })} value={route.name} /></label>
      <label><input checked={route.reaches_home} onChange={(e) => patchRoute(route.id, { reaches_home: e.target.checked })} type="checkbox" />Can reach HOME</label>
      <label><input checked={route.enabled} onChange={(e) => patchRoute(route.id, { enabled: e.target.checked })} type="checkbox" />Route enabled</label>
      <label><input checked={route.one_way} onChange={(e) => patchRoute(route.id, { one_way: e.target.checked })} type="checkbox" />One-way transfer</label>
      <label>Available from<input onChange={(e) => patchRoute(route.id, { starts_at: e.target.value ? new Date(e.target.value).toISOString() : null })} type="datetime-local" value={localDate(route.starts_at)} /></label>
      <label>Unavailable from<input onChange={(e) => patchRoute(route.id, { ends_at: e.target.value ? new Date(e.target.value).toISOString() : null })} type="datetime-local" value={localDate(route.ends_at)} /></label>
    </article>)}</div><button className="secondary" onClick={addRoute}>Add a transfer route</button></fieldset>
    <p className="catalogue-hint">Bank’s initial end date comes from <a href="https://en-americas-support.nintendo.com/app/answers/detail/a_id/61543/" rel="noreferrer" target="_blank">Nintendo’s shutdown notice</a>. You can change it here if the announcement changes.</p>
    </section>

    <section className="catalogue-card"><h2>Encounter editor</h2><p>Your edits sit on top of imported information. Future reference imports won’t overwrite them.</p><div className="catalogue-grid">
      <label>Find Pokémon<input onChange={(e) => setPokemonSearch(e.target.value)} placeholder="Name or National Dex number" value={pokemonSearch} /></label>
      <label>Pokémon<select onChange={(e) => setSelectedPokemon(Number(e.target.value))} value={selectedPokemon}>{pokemon.filter((mon) => mon.id === selectedPokemon || `${mon.name} ${mon.national_id} ${mon.form || ''}`.toLowerCase().includes(pokemonSearch.toLowerCase())).map((mon) => <option key={mon.id} value={mon.id}>#{mon.national_id} {mon.name}{mon.form ? ` (${mon.form})` : ''}</option>)}</select></label>
      <label>Source game<select onChange={(e) => setSelectedGame(e.target.value)} value={selectedGame}>{data.settings.games.filter((game) => game.id !== 'home').map((game) => <option key={game.id} value={game.id}>{game.name}</option>)}</select></label>
    </div><p className="catalogue-hint">Choose Keep my encounter text, then Save changes to apply your edit.</p><fieldset disabled={!editable || busy || encounterLoading}><label>Where to find {selectedMon?.name || 'this Pokémon'} — one location or instruction per line<textarea onChange={(e) => setEncounterText(e.target.value)} placeholder="Enter a location, gift, trade, or transfer instruction" rows={6} value={encounterText} /></label>
      <div className="catalogue-actions"><button onClick={keepEncounter}>Keep my encounter text</button><button className="secondary" disabled={!selectedOverride} onClick={restoreEncounter}>Restore imported</button><span>{selectedOverride ? 'Personal override' : importedLines.length ? 'Imported information' : 'No imported encounter for this game'}</span></div></fieldset>
    <details><summary>See the original imported text</summary><pre>{importedLines.join('\n') || 'No imported encounter for this game.'}</pre></details><p className="catalogue-hint">Keeping an empty override hides this game’s encounter for the selected Pokémon. That doesn’t remove the Pokémon from any dex.</p>
    </section>

    <section className="catalogue-card"><h2>Take your configuration with you</h2><p>Export games, source rules, and personal encounter edits. Passwords and caught progress are excluded.</p><div className="catalogue-actions"><button className="secondary" onClick={exportConfiguration}>Export configuration</button><label>Import configuration<input accept=".json,application/json" disabled={!editable} onChange={(e) => void readImport(e.target.files?.[0])} type="file" /></label></div>
      {pendingImport && <div className="catalogue-notice"><p>Preview: {pendingImport.settings.games.length} games, {pendingImport.settings.routes.length} routes, {pendingImport.overrides.length} encounter overrides. This replaces your editable configuration, while retaining imported data and caught progress.</p><button disabled={!editable} onClick={() => {
        setData({ ...pendingImport, revision: data.revision, is_owner: editable }); setPendingImport(null); setNotice('Configuration loaded into your changes. Review it, then Save changes.');
      }}>Use this configuration</button><button className="secondary" onClick={() => setPendingImport(null)}>Cancel</button></div>}
    </section>
  </main></>;
}
