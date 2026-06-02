# MS-ODRAWXML Reference Notes

This directory tracks Office Drawing extension references that appear in real
fixtures but are outside the ECMA-376 Strict schema bundle.

Source:

- Microsoft Learn, `[MS-ODRAWXML]: Office Drawing Extensions to Office Open XML Structure`
- https://learn.microsoft.com/en-us/openspecs/office_standards/ms-odrawxml/

Important anchors for current renderer work:

- `useLocalDpi` extension element:
  https://learn.microsoft.com/en-us/openspecs/office_standards/ms-odrawxml/c05c287f-f63e-4fa9-8163-e3e106b4105f
- Office Drawing 2010 main schema, including `hiddenFill` and `CT_UseLocalDpi`:
  https://learn.microsoft.com/en-us/openspecs/office_standards/ms-odrawxml/869af0ef-665b-4f28-a596-0917474ede26

Maintenance rules:

- Treat ECMA-376 as the primary DrawingML model.
- Use MS-ODRAWXML only for `a14`/Microsoft extension elements preserved in
  source XML.
- Do not turn an extension into renderer behavior unless the source object,
  official extension documentation, and an attributed object fixture all support
  that interpretation.
