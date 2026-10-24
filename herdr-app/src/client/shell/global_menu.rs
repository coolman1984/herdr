use super::*;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub(super) enum ClientGlobalMenuAction {
    Binding(crate::input::KeybindAction),
    WhatsNew,
}

pub(super) fn global_menu_attention(snapshot: &ClientShellSnapshot) -> bool {
    snapshot.update_available.is_some() || snapshot.integration_updates_available
}

pub(super) fn global_menu_item_has_badge(
    snapshot: &ClientShellSnapshot,
    action: ClientGlobalMenuAction,
) -> bool {
    (action == ClientGlobalMenuAction::WhatsNew && snapshot.update_available.is_some())
        || (action == ClientGlobalMenuAction::Binding(crate::input::KeybindAction::Settings)
            && snapshot.integration_updates_available)
}

pub(super) fn global_menu_items(
    snapshot: &ClientShellSnapshot,
) -> Vec<(&'static str, ClientGlobalMenuAction)> {
    let mut items = vec![
        (
            "settings",
            ClientGlobalMenuAction::Binding(crate::input::KeybindAction::Settings),
        ),
        (
            "keybinds",
            ClientGlobalMenuAction::Binding(crate::input::KeybindAction::Help),
        ),
        (
            "reload config",
            ClientGlobalMenuAction::Binding(crate::input::KeybindAction::ReloadConfig),
        ),
    ];
    if snapshot.update_available.is_some() || snapshot.latest_release_notes_available {
        items.push((
            if snapshot.update_available.is_some() {
                "update ready"
            } else {
                "what's new"
            },
            ClientGlobalMenuAction::WhatsNew,
        ));
    }
    items.push((
        "detach",
        ClientGlobalMenuAction::Binding(crate::input::KeybindAction::Detach),
    ));
    items
}

impl ClientShellState {
    /// Local custom commands that set `menu = "..."`, each paired with the live
    /// command its key runs. Menus list them after their built-in items.
    pub(super) fn custom_menu_entries(&self) -> Vec<(String, crate::config::CustomCommandKeybind)> {
        let commands = &self.config.keybinds.keybinds.custom_commands;
        self.config
            .local_keys
            .command
            .iter()
            .filter_map(|configured| {
                let menu = configured.menu.as_deref()?.trim();
                if menu.is_empty() {
                    return None;
                }
                let labels: Vec<String> = configured
                    .key
                    .values()
                    .into_iter()
                    .map(str::to_owned)
                    .collect();
                let key_label = crate::config::ActionKeybinds::from_labels(&labels)
                    .ok()?
                    .label()?;
                let command = commands
                    .iter()
                    .find(|command| command.bindings.label().as_deref() == Some(key_label.as_str()))?;
                Some((menu.to_owned(), command.clone()))
            })
            .collect()
    }

    pub(super) fn custom_menu_labels(&self) -> Vec<String> {
        self.custom_menu_entries()
            .into_iter()
            .map(|(label, _)| label)
            .collect()
    }

    /// Runs the custom menu entry at `index`; false when there is none.
    pub(super) fn activate_custom_menu_entry(
        &mut self,
        index: usize,
        outcome: &mut ClientShellInput,
    ) -> bool {
        let Some((_, command)) = self.custom_menu_entries().into_iter().nth(index) else {
            return false;
        };
        self.record_binding(crate::input::KeybindMatch::Command(command), outcome);
        true
    }

    pub(super) fn toggle_global_menu(&mut self) {
        if matches!(self.overlay, Some(ClientShellOverlay::GlobalMenu(_))) {
            self.overlay = None;
        } else {
            self.overlay = Some(ClientShellOverlay::GlobalMenu(ClientGlobalMenuOverlay {
                highlighted: 0,
            }));
        }
    }

    pub(super) fn move_global_menu_selection(&mut self, delta: isize) {
        let item_count = self
            .snapshot
            .as_deref()
            .map(global_menu_items)
            .map_or(0, |items| items.len())
            + self.custom_menu_entries().len();
        let Some(ClientShellOverlay::GlobalMenu(menu)) = self.overlay.as_mut() else {
            return;
        };
        menu.highlighted = (menu.highlighted as isize + delta)
            .clamp(0, item_count.saturating_sub(1) as isize) as usize;
    }

    pub(super) fn activate_global_menu_item(
        &mut self,
        index: usize,
        outcome: &mut ClientShellInput,
    ) {
        let Some(builtin_count) = self
            .snapshot
            .as_deref()
            .map(|snapshot| global_menu_items(snapshot).len())
        else {
            return;
        };
        if index >= builtin_count {
            self.overlay = None;
            self.activate_custom_menu_entry(index - builtin_count, outcome);
            outcome.repaint = true;
            return;
        }
        let Some(action) = self.snapshot.as_deref().and_then(|snapshot| {
            global_menu_items(snapshot)
                .get(index)
                .map(|(_, action)| *action)
        }) else {
            return;
        };
        if action == ClientGlobalMenuAction::WhatsNew
            && self
                .snapshot
                .as_deref()
                .and_then(|snapshot| snapshot.release_notes.as_ref())
                .is_none()
        {
            return;
        }
        self.overlay = None;
        match action {
            ClientGlobalMenuAction::Binding(binding) => {
                self.record_binding(crate::input::KeybindMatch::Action(binding), outcome)
            }
            ClientGlobalMenuAction::WhatsNew => self.open_release_notes(),
        }
        outcome.repaint = true;
    }
}
