# Building workflows in Ticketbot

One page for people who build and edit workflows. The same text is behind the ⓘ buttons in the
editor.

## What a workflow is

A workflow belongs to one ConnectWise board. Every ticket event on that board, a ticket being
created or updated, enters the workflow at each **Trigger** that listens for that event and walks
the wires from there. A trigger can carry an **Only when** condition, built the same way as an
If; when it does not hold, that lane does not run and the run's history says so. Give each rule
its own trigger, a lane, rather than hanging rules off one another. The canvas is the workflow: steps are cards, wires are the paths between
them.

## How a walk moves

- An **If** step leaves by its **match** port when its condition holds and by **else** when it
  does not. A port with nothing wired to it ends that path.
- A step that two paths both reach runs once.
- **Skip notify** silences the Notify steps after it on its own path only.
- Ticket writes (status, priority, owner, resources, patch) are collected and sent to ConnectWise
  as one change after the whole flow has run. If two steps set the same field, the later one wins
  and the ticket history lists the conflict. Notes are added after that change; notifications go
  out last.

## Moving steps around

Right-click a step for Duplicate, Copy, Enable or Disable and Delete. Shift-drag on empty canvas
selects several steps at once, and Shift-click adds one to the selection; dragging any selected
step moves them all. Copied steps can be pasted into another board's workflow with the wires
between them intact, by right-clicking the empty canvas. Keyboard: ⌘C copies, ⌘V pastes, ⌘D
duplicates, Delete removes.

## Conditions

Each row is a field, a comparison and a value. **Match all** means every row must hold; **match
any** means one is enough.

- **is / is not** hold on every update while the field has (or lacks) that value.
- **changed to** holds only on the update that moved the field to that value, so "Status changed
  to Waiting" fires once, when it happens.
- **changed from** matches the value the field left.
- **contains** matches text anywhere in the field. **is in list** checks membership of a list you
  keep under Lists (companies or contacts).

**Advanced** shows the same condition as text and accepts anything the engine understands,
including parentheses and `not`. A condition the builder cannot show opens in Advanced with a
note saying why. Use **Validate** to check the text and **Test** to evaluate it against a ticket.

## Messages

A Notify step sends the default layout unless you give it a custom message. On an update the
default layout includes a **Changed** line (for example `Status: New → Assigned`) so two updates in
a row read differently; `{{changes}}` puts the same text in a custom message. A step whose people
list comes out empty, because the only person on the ticket wrote the note, shows "Nobody to
notify" in the ticket history instead of sending. Click a `{{token}}` to
insert it at the cursor. **Preview** renders the message with a real ticket so you can see what
each token produces. Notifications go to a room, a person, or the ticket's resources and owner
(skipping whoever wrote the note that triggered the run).

## Seeing what ran

The **Results** tab beside the workflow list shows every run: when, which ticket, what fired and
how it ended. Open a run to see the path it took drawn on the workflow's canvas, then every step,
write and notification underneath. Filter by board, outcome, date or ticket; the **Results** button
in a workflow's toolbar opens the list already filtered to that board.

## Trying it safely

1. Turn on the workflow's **Dry run**. The flow runs and records what it would have done in the
   ticket history (Tickets page), but writes nothing to ConnectWise and sends nothing.
2. Use **Simulate** in the toolbar with a real ticket number. The path lights up on the canvas and
   the panel lists who would be notified and with what message.
3. When the history looks right, turn dry run off and save.

An administrator sets the business hours under Config; the "no webhooks" alert only fires inside
them. An administrator can also put the whole instance in **master dry run** (Config), which overrides
every workflow. While notifications are **redirected** to a test room (Config), you will see the
real messages there instead of in their intended rooms.

## Connecting Claude

Ticketbot can be added to Claude (Claude.ai, Claude Desktop or Claude Code) as a connector. Claude
then has tools to read your workflows, what each run did, the history ticketbot kept for a ticket,
lists, forwards and settings, and to simulate a workflow or test a condition against a stored
ticket. Every tool is read-only: Claude cannot change a workflow or write to ConnectWise or Webex
through ticketbot.

1. Open **Connected apps** in the account menu and copy the connector URL.
2. In Claude, add a custom connector with that URL.
3. Claude opens a Ticketbot sign-in. Sign in as usual (including Microsoft or your authenticator
   code), read what the client is asking for and choose **Allow**.

Claude only ever sees what your role already allows: a viewer's Claude cannot see the intake
queue, an administrator's can. Each connection is listed under **Connected apps** with when it was
made and last used; **Disconnect** revokes it at once, and connecting again from Claude restores
it. An administrator can disconnect anyone's from the Users page, and turns the whole feature on
or off with **MCP server** under Config.

Ticketbot's tools are named `ticketbot_…` so they never collide with a ConnectWise connector's
`cw_…` tools. Ask Claude about ticketbot's workflows and history here, and about live ticket data
through ConnectWise; the ids are the same in both.
