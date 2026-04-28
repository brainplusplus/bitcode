# bc-dialog-wizard

> Multi-step wizard dialog

## Quick Start

```html
<bc-dialog-wizard></bc-dialog-wizard>
```

## Props

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| open | boolean | false | Open state |
| dialog-title | string | '' | Title |
| steps | string (JSON) | '[]' | Array of step labels |

## Events

| Event | Payload | Description |
|-------|---------|-------------|
| lcDialogClose | {type, step} | Closed |
| lcWizardComplete | {step} | All steps completed |

## Methods

| Method | Returns | Description |
|--------|---------|-------------|
| openDialog() | Promise<void> | Open |
| closeDialog() | Promise<void> | Close |
| goToStep(n) | Promise<void> | Jump to step |
| nextStep() | Promise<void> | Next step |
| prevStep() | Promise<void> | Previous step |
| getCurrentStep() | Promise<number> | Get current step index |

See [theming](../theming.md).

