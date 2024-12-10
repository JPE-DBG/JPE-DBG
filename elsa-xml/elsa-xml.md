We need libyml2 library in Linux (b/o cgo). To check it run in cmd as follows:

```
pkg-config --list-all | grep libxml

It should print:
libxml-2.0     libXML - libXML library version2.
```

Folders “testdata” and “hold” contain what the name suggests, depending on where you unpack those to, you will have to set the following environment variables accordingly:

- SCHEMA_DIR_T2S => {wherever you unpacked to}/schemas/T2S
- SCHEMA_DIR_ISO => {wherever you unpacked to}/schemas/ISO
  
  note: only one needs to be set, if both are missing, program will exit with an error

- TEST_DATA_DIR => 
  - {wherever you unpacked to}/schemas/T2S/testdata/CREA (holds only pure ISO format message as we will receive from CREATION)
  - {wherever you unpacked to}/schemas/T2S/testdata/T2S (holds only 20022+ format message as we will receive from T2S/PMCSD)
  - {wherever you unpacked to}/schemas/T2S/testdata/full (combination of those above)
