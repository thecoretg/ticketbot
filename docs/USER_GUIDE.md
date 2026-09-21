# Building workflows in Ticketbot

One page for people who build and edit workflows. The same text is behind the ⓘ buttons in the
editor.

## What a workflow is

A workflow belongs to one ConnectWise board. Every ticket event on that board, a ticket being
created or updated, enters the workflow at each **Trigger** that listens for that event and walks
the wires from there. The canvas is the workflow: steps are cards, wires are the paths between
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

A Notify step sends the default layout unless you give it a custom message. Click a `{{token}}` to
insert it at the cursor. **Preview** renders the message with a real ticket so you can see what
each token produces. Notifications go to a room, a person, or the ticket's resources and owner
(skipping whoever wrote the note that triggered the run).

## Trying it safely

1. Turn on the workflow's **Dry run**. The flow runs and records what it would have done in the
   ticket history (Tickets page), but writes nothing to ConnectWise and sends nothing.
2. Use **Simulate** in the toolbar with a real ticket number. The path lights up on the canvas and
   the panel lists who would be notified and with what message.
3. When the history looks right, turn dry run off and save.

An administrator can also put the whole instance in **master dry run** (Config), which overrides
every workflow. While notifications are **redirected** to a test room (Config), you will see the
real messages there instead of in their intended rooms.
