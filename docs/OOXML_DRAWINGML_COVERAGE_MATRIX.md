# OOXML/DrawingML Coverage Matrix

This is the exhaustive schema-declaration audit for Puppt's static PPTX renderer target.
It is generated from the local ECMA-376 strict-schema files under
`docs/specs/ecma-376/part1/schema/strict/`.

Scope for this audit:

- `pml.xsd`
- every `dml-*.xsd` file in the local strict schema bundle
- every top-level `xsd:complexType`, `xsd:simpleType`, `xsd:group`,
  `xsd:attributeGroup`, `xsd:element`, and `xsd:attribute` declaration

This makes the audit complete over the PresentationML/DrawingML schema
declarations available in the repo. Nested child elements and attributes are
summarized in the `Members` column of their owning declaration instead of being
duplicated as separate rows.

Status definitions:

- **Supported**: current code/docs provide explicit evidence for the covered
  declaration subset.
- **Partial**: current code/docs cover some semantics, but known child elements,
  attributes, renderer behavior, or fixtures remain incomplete.
- **Unsupported**: currently not rendered or only reported/preserved.
- **Out of renderer scope**: schema belongs to another host application or a
  non-static-rendering area outside the current Puppt renderer goal.
- **Unimplemented / no evidence**: the declaration exists in the spec but has no
  maintained code/doc/test evidence yet, so it is treated as unimplemented
  until proven otherwise.

## Audit Totals

- Total schema declarations audited: **1007**

| Status | Count |
|---|---:|
| Supported | 16 |
| Partial | 178 |
| Unsupported | 444 |
| Out of renderer scope | 74 |
| Unimplemented / no evidence | 295 |

## File Totals

| Schema file | Declarations | Supported | Partial | Unsupported | Out of scope | Unimplemented/no evidence |
|---|---:|---:|---:|---:|---:|---:|
| `pml.xsd` | 226 | 5 | 42 | 58 | 16 | 105 |
| `dml-main.xsd` | 354 | 11 | 130 | 23 | 0 | 190 |
| `dml-picture.xsd` | 3 | 0 | 3 | 0 | 0 | 0 |
| `dml-diagram.xsd` | 138 | 0 | 3 | 135 | 0 | 0 |
| `dml-chart.xsd` | 210 | 0 | 0 | 210 | 0 | 0 |
| `dml-chartDrawing.xsd` | 17 | 0 | 0 | 17 | 0 | 0 |
| `dml-lockedCanvas.xsd` | 1 | 0 | 0 | 1 | 0 | 0 |
| `dml-spreadsheetDrawing.xsd` | 25 | 0 | 0 | 0 | 25 | 0 |
| `dml-wordprocessingDrawing.xsd` | 33 | 0 | 0 | 0 | 33 | 0 |

## Coverage Rows

Each row is a schema declaration. The anchor line points at the declaration in
the local schema file.

### pml.xsd

| Anchor | Kind | Declaration | Members | Status | Evidence / gap |
|---|---|---|---|---|---|
| `pml.xsd:14` | simpleType | `ST_TransitionSideDirectionType` | - | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:22` | simpleType | `ST_TransitionCornerDirectionType` | - | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:30` | simpleType | `ST_TransitionInOutDirectionType` | - | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:36` | complexType | `CT_SideDirectionTransition` | attr:dir | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:39` | complexType | `CT_CornerDirectionTransition` | attr:dir | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:42` | simpleType | `ST_TransitionEightDirectionType` | - | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:45` | complexType | `CT_EightDirectionTransition` | attr:dir | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:48` | complexType | `CT_OrientationTransition` | attr:dir | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:51` | complexType | `CT_InOutTransition` | attr:dir | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:54` | complexType | `CT_OptionalBlackTransition` | attr:thruBlk | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:57` | complexType | `CT_SplitTransition` | attr:orient, attr:dir | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:61` | complexType | `CT_WheelTransition` | attr:spokes | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:64` | complexType | `CT_TransitionStartSoundAction` | el:snd, attr:loop | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:70` | complexType | `CT_TransitionSoundAction` | el:stSnd, el:endSnd | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:76` | simpleType | `ST_TransitionSpeed` | - | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:83` | complexType | `CT_SlideTransition` | el:blinds, el:checker, el:circle, el:dissolve, el:comb, el:cover, el:cut, el:diamond ... | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:115` | simpleType | `ST_TLTimeIndefinite` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:120` | simpleType | `ST_TLTime` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:123` | simpleType | `ST_TLTimeNodeID` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:126` | complexType | `CT_TLIterateIntervalTime` | attr:val | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:129` | complexType | `CT_TLIterateIntervalPercentage` | attr:val | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:132` | simpleType | `ST_IterateType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:139` | complexType | `CT_TLIterateData` | el:tmAbs, el:tmPct, attr:type, attr:backwards | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:147` | complexType | `CT_TLSubShapeId` | attr:spid | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:150` | complexType | `CT_TLTextTargetElement` | el:charRg, el:pRg | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:156` | simpleType | `ST_TLChartSubelementType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:165` | complexType | `CT_TLOleChartTargetElement` | attr:type, attr:lvl | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:169` | complexType | `CT_TLShapeTargetElement` | el:bg, el:subSp, el:oleChartEl, el:txEl, el:graphicEl, attr:spid | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:179` | complexType | `CT_TLTimeTargetElement` | el:sldTgt, el:sndTgt, el:spTgt, el:inkTgt | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:187` | complexType | `CT_TLTriggerTimeNodeID` | attr:val | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:190` | simpleType | `ST_TLTriggerRuntimeNode` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:197` | complexType | `CT_TLTriggerRuntimeNode` | attr:val | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:200` | simpleType | `ST_TLTriggerEvent` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:215` | complexType | `CT_TLTimeCondition` | el:tgtEl, el:tn, el:rtn, attr:evt, attr:delay | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:224` | complexType | `CT_TLTimeConditionList` | el:cond | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:229` | complexType | `CT_TimeNodeList` | el:par, el:seq, el:excl, el:anim, el:animClr, el:animEffect, el:animMotion, el:animRot ... | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:246` | simpleType | `ST_TLTimeNodePresetClassType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:256` | simpleType | `ST_TLTimeNodeRestartType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:263` | simpleType | `ST_TLTimeNodeFillType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:271` | simpleType | `ST_TLTimeNodeSyncType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:277` | simpleType | `ST_TLTimeNodeMasterRelation` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:284` | simpleType | `ST_TLTimeNodeType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:297` | complexType | `CT_TLCommonTimeNodeData` | el:stCondLst, el:endCondLst, el:endSync, el:iterate, el:childTnLst, el:subTnLst, attr:id, attr:presetID ... | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:330` | complexType | `CT_TLTimeNodeParallel` | el:cTn | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:335` | simpleType | `ST_TLNextActionType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:341` | simpleType | `ST_TLPreviousActionType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:347` | complexType | `CT_TLTimeNodeSequence` | el:cTn, el:prevCondLst, el:nextCondLst, attr:concurrent, attr:prevAc, attr:nextAc | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:357` | complexType | `CT_TLTimeNodeExclusive` | el:cTn | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:362` | complexType | `CT_TLBehaviorAttributeNameList` | el:attrName | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:367` | simpleType | `ST_TLBehaviorAdditiveType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:376` | simpleType | `ST_TLBehaviorAccumulateType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:382` | simpleType | `ST_TLBehaviorTransformType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:388` | simpleType | `ST_TLBehaviorOverrideType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:394` | complexType | `CT_TLCommonBehaviorData` | el:cTn, el:tgtEl, el:attrNameLst, attr:additive, attr:accumulate, attr:xfrmType, attr:from, attr:to ... | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:410` | complexType | `CT_TLAnimVariantBooleanVal` | attr:val | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:413` | complexType | `CT_TLAnimVariantIntegerVal` | attr:val | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:416` | complexType | `CT_TLAnimVariantFloatVal` | attr:val | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:419` | complexType | `CT_TLAnimVariantStringVal` | attr:val | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:422` | complexType | `CT_TLAnimVariant` | el:boolVal, el:intVal, el:fltVal, el:strVal, el:clrVal | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:431` | simpleType | `ST_TLTimeAnimateValueTime` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:434` | complexType | `CT_TLTimeAnimateValue` | el:val, attr:tm, attr:fmla | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:441` | complexType | `CT_TLTimeAnimateValueList` | el:tav | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:446` | simpleType | `ST_TLAnimateBehaviorCalcMode` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:453` | simpleType | `ST_TLAnimateBehaviorValueType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:460` | complexType | `CT_TLAnimateBehavior` | el:cBhvr, el:tavLst, attr:by, attr:from, attr:to, attr:calcmode, attr:valueType | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:471` | complexType | `CT_TLByRgbColorTransform` | attr:r, attr:g, attr:b | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:476` | complexType | `CT_TLByHslColorTransform` | attr:h, attr:s, attr:l | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:481` | complexType | `CT_TLByAnimateColorTransform` | el:rgb, el:hsl | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:487` | simpleType | `ST_TLAnimateColorSpace` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:493` | simpleType | `ST_TLAnimateColorDirection` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:499` | complexType | `CT_TLAnimateColorBehavior` | el:cBhvr, el:by, el:from, el:to, attr:clrSpc, attr:dir | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:509` | simpleType | `ST_TLAnimateEffectTransition` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:516` | complexType | `CT_TLAnimateEffectBehavior` | el:cBhvr, el:progress, attr:transition, attr:filter, attr:prLst | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:525` | simpleType | `ST_TLAnimateMotionBehaviorOrigin` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:531` | simpleType | `ST_TLAnimateMotionPathEditMode` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:537` | complexType | `CT_TLPoint` | attr:x, attr:y | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:541` | complexType | `CT_TLAnimateMotionBehavior` | el:cBhvr, el:by, el:from, el:to, el:rCtr, attr:origin, attr:path, attr:pathEditMode ... | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:555` | complexType | `CT_TLAnimateRotationBehavior` | el:cBhvr, attr:by, attr:from, attr:to | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:563` | complexType | `CT_TLAnimateScaleBehavior` | el:cBhvr, el:by, el:from, el:to, attr:zoomContents | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:572` | simpleType | `ST_TLCommandType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:579` | complexType | `CT_TLCommandBehavior` | el:cBhvr, attr:type, attr:cmd | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:586` | complexType | `CT_TLSetBehavior` | el:cBhvr, el:to | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:592` | complexType | `CT_TLCommonMediaNodeData` | el:cTn, el:tgtEl, attr:vol, attr:mute, attr:numSld, attr:showWhenStopped | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:602` | complexType | `CT_TLMediaNodeAudio` | el:cMediaNode, attr:isNarration | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:608` | complexType | `CT_TLMediaNodeVideo` | el:cMediaNode, attr:fullScrn | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:614` | attributeGroup | `AG_TLBuild` | attr:spid, attr:grpId, attr:uiExpand | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:619` | complexType | `CT_TLTemplate` | el:tnLst, attr:lvl | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:625` | complexType | `CT_TLTemplateList` | el:tmpl | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:630` | simpleType | `ST_TLParaBuildType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:638` | complexType | `CT_TLBuildParagraph` | el:tmplLst, attr:build, attr:bldLvl, attr:animBg, attr:autoUpdateAnimBg, attr:rev, attr:advAuto | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:650` | simpleType | `ST_TLDiagramBuildType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:671` | complexType | `CT_TLBuildDiagram` | attr:bld | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:675` | simpleType | `ST_TLOleChartBuildType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:684` | complexType | `CT_TLOleBuildChart` | attr:bld, attr:animBg | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:689` | complexType | `CT_TLGraphicalObjectBuild` | el:bldAsOne, el:bldSub | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:696` | complexType | `CT_BuildList` | el:bldP, el:bldDgm, el:bldOleChart, el:bldGraphic | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:704` | complexType | `CT_SlideTiming` | el:tnLst, el:bldLst, el:extLst | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:711` | complexType | `CT_Empty` | - | Supported | Covered by package, presentation, slide-order, or slide-size workflows. |
| `pml.xsd:712` | simpleType | `ST_Name` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:715` | simpleType | `ST_Direction` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:721` | simpleType | `ST_Index` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:724` | complexType | `CT_IndexRange` | attr:st, attr:end | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:728` | complexType | `CT_SlideRelationshipListEntry` | attr:r:id | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:731` | complexType | `CT_SlideRelationshipList` | el:sld | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:737` | complexType | `CT_CustomShowId` | attr:id | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:740` | group | `EG_SlideListChoice` | el:sldAll, el:sldRg, el:custShow | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:747` | complexType | `CT_CustomerData` | attr:r:id | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:750` | complexType | `CT_TagsData` | attr:r:id | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:753` | complexType | `CT_CustomerDataList` | el:custData, el:tags | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:759` | complexType | `CT_Extension` | attr:uri | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:765` | group | `EG_ExtensionList` | el:ext | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:770` | complexType | `CT_ExtensionList` | group:EG_ExtensionList | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:775` | complexType | `CT_ExtensionListModify` | group:EG_ExtensionList, attr:mod | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:781` | complexType | `CT_CommentAuthor` | el:extLst, attr:id, attr:name, attr:initials, attr:lastIdx, attr:clrIdx | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:791` | complexType | `CT_CommentAuthorList` | el:cmAuthor | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:796` | element | `cmAuthorLst` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:797` | complexType | `CT_Comment` | el:pos, el:text, el:extLst, attr:authorId, attr:dt, attr:idx | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:807` | complexType | `CT_CommentList` | el:cm | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:812` | element | `cmLst` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:813` | attributeGroup | `AG_Ole` | attr:name, attr:showAsIcon, attr:r:id, attr:imgW, attr:imgH | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:820` | simpleType | `ST_OleObjectFollowColorScheme` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:827` | complexType | `CT_OleObjectEmbed` | el:extLst, attr:followColorScheme | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:834` | complexType | `CT_OleObjectLink` | el:extLst, attr:updateAutomatic | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:840` | complexType | `CT_OleObject` | el:embed, el:link, el:pic, attr:progId | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:851` | element | `oleObj` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:852` | complexType | `CT_Control` | el:extLst, el:pic | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:859` | complexType | `CT_ControlList` | el:control | Unsupported | Detected only partially or not rendered for static output. |
| `pml.xsd:864` | simpleType | `ST_SlideId` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:870` | complexType | `CT_SlideIdListEntry` | el:extLst, attr:id, attr:r:id | Supported | Covered by package, presentation, slide-order, or slide-size workflows. |
| `pml.xsd:877` | complexType | `CT_SlideIdList` | el:sldId | Supported | Covered by package, presentation, slide-order, or slide-size workflows. |
| `pml.xsd:882` | simpleType | `ST_SlideMasterId` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:887` | complexType | `CT_SlideMasterIdListEntry` | el:extLst, attr:id, attr:r:id | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:894` | complexType | `CT_SlideMasterIdList` | el:sldMasterId | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:900` | complexType | `CT_NotesMasterIdListEntry` | el:extLst, attr:r:id | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:906` | complexType | `CT_NotesMasterIdList` | el:notesMasterId | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:912` | complexType | `CT_HandoutMasterIdListEntry` | el:extLst, attr:r:id | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:918` | complexType | `CT_HandoutMasterIdList` | el:handoutMasterId | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:924` | complexType | `CT_EmbeddedFontDataId` | attr:r:id | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:927` | complexType | `CT_EmbeddedFontListEntry` | el:font, el:regular, el:bold, el:italic, el:boldItalic | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:936` | complexType | `CT_EmbeddedFontList` | el:embeddedFont | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:942` | complexType | `CT_SmartTags` | attr:r:id | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:945` | complexType | `CT_CustomShow` | el:sldLst, el:extLst, attr:name, attr:id | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:953` | complexType | `CT_CustomShowList` | el:custShow | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:958` | simpleType | `ST_PhotoAlbumLayout` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:969` | simpleType | `ST_PhotoAlbumFrameShape` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:980` | complexType | `CT_PhotoAlbum` | el:extLst, attr:bw, attr:showCaptions, attr:layout, attr:frame | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:990` | simpleType | `ST_SlideSizeCoordinate` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:996` | simpleType | `ST_SlideSizeType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1016` | complexType | `CT_SlideSize` | attr:cx, attr:cy, attr:type | Supported | Covered by package, presentation, slide-order, or slide-size workflows. |
| `pml.xsd:1021` | complexType | `CT_Kinsoku` | attr:lang, attr:invalStChars, attr:invalEndChars | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:1026` | simpleType | `ST_BookmarkIdSeed` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1032` | complexType | `CT_ModifyVerifier` | attr:algorithmName, attr:hashValue, attr:saltValue, attr:spinValue | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:1038` | complexType | `CT_Presentation` | el:sldMasterIdLst, el:notesMasterIdLst, el:handoutMasterIdLst, el:sldIdLst, el:sldSz, el:notesSz, el:smartTags, el:embeddedFontLst ... | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1070` | element | `presentation` | - | Supported | Covered by package, presentation, slide-order, or slide-size workflows. |
| `pml.xsd:1071` | complexType | `CT_HtmlPublishProperties` | group:EG_SlideListChoice, el:extLst, attr:showSpeakerNotes, attr:target, attr:title, attr:r:id | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1081` | simpleType | `ST_PrintWhat` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1094` | simpleType | `ST_PrintColorMode` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1101` | complexType | `CT_PrintProperties` | el:extLst, attr:prnWhat, attr:clrMode, attr:hiddenSlides, attr:scaleToFitPaper, attr:frameSlides | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:1111` | complexType | `CT_ShowInfoBrowse` | attr:showScrollbar | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:1114` | complexType | `CT_ShowInfoKiosk` | attr:restart | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:1117` | group | `EG_ShowType` | el:present, el:browse, el:kiosk | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1124` | complexType | `CT_ShowProperties` | group:EG_ShowType, group:EG_SlideListChoice, el:penClr, el:extLst, attr:loop, attr:showNarration, attr:showAnimation, attr:useTimings ... | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:1136` | complexType | `CT_PresentationProperties` | el:prnPr, el:showPr, el:clrMru, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1144` | element | `presentationPr` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1145` | complexType | `CT_HeaderFooter` | el:extLst, attr:sldNum, attr:hdr, attr:ftr, attr:dt | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1154` | simpleType | `ST_PlaceholderType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1174` | simpleType | `ST_PlaceholderSize` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1181` | complexType | `CT_Placeholder` | el:extLst, attr:type, attr:orient, attr:sz, attr:idx, attr:hasCustomPrompt | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1191` | complexType | `CT_ApplicationNonVisualDrawingProps` | el:ph, group:a:EG_Media, el:custDataLst, el:extLst, attr:isPhoto, attr:userDrawn | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1201` | complexType | `CT_ShapeNonVisual` | el:cNvPr, el:cNvSpPr, el:nvPr | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1209` | complexType | `CT_Shape` | el:nvSpPr, el:spPr, el:style, el:txBody, el:extLst, attr:useBgFill | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1219` | complexType | `CT_ConnectorNonVisual` | el:cNvPr, el:cNvCxnSpPr, el:nvPr | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1228` | complexType | `CT_Connector` | el:nvCxnSpPr, el:spPr, el:style, el:extLst | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1236` | complexType | `CT_PictureNonVisual` | el:cNvPr, el:cNvPicPr, el:nvPr | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1245` | complexType | `CT_Picture` | el:nvPicPr, el:blipFill, el:spPr, el:style, el:extLst | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1254` | complexType | `CT_GraphicalObjectFrameNonVisual` | el:cNvPr, el:cNvGraphicFramePr, el:nvPr | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1263` | complexType | `CT_GraphicalObjectFrame` | el:nvGraphicFramePr, el:xfrm, el:a:graphic, el:extLst, attr:bwMode | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1273` | complexType | `CT_GroupShapeNonVisual` | el:cNvPr, el:cNvGrpSpPr, el:nvPr | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1282` | complexType | `CT_GroupShape` | el:nvGrpSpPr, el:grpSpPr, el:sp, el:grpSp, el:graphicFrame, el:cxnSp, el:pic, el:contentPart ... | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1297` | complexType | `CT_Rel` | attr:r:id | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1300` | group | `EG_TopLevelSlide` | el:clrMap | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1305` | group | `EG_ChildSlide` | el:clrMapOvr | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1310` | attributeGroup | `AG_ChildSlide` | attr:showMasterSp, attr:showMasterPhAnim | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1314` | complexType | `CT_BackgroundProperties` | group:a:EG_FillProperties, group:a:EG_EffectProperties, el:extLst, attr:shadeToTitle | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1322` | group | `EG_Background` | el:bgPr, el:bgRef | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1328` | complexType | `CT_Background` | group:EG_Background, attr:bwMode | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1334` | complexType | `CT_CommonSlideData` | el:bg, el:spTree, el:custDataLst, el:controls, el:extLst, attr:name | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1344` | complexType | `CT_Slide` | el:cSld, group:EG_ChildSlide, el:transition, el:timing, el:extLst, attr:show | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:733` | element | `sld` | - | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1356` | simpleType | `ST_SlideLayoutType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1396` | complexType | `CT_SlideLayout` | el:cSld, group:EG_ChildSlide, el:transition, el:timing, el:hf, el:extLst, attr:matchingName, attr:type ... | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1411` | element | `sldLayout` | - | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1412` | complexType | `CT_SlideMasterTextStyles` | el:titleStyle, el:bodyStyle, el:otherStyle, el:extLst | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1420` | simpleType | `ST_SlideLayoutId` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1425` | complexType | `CT_SlideLayoutIdListEntry` | el:extLst, attr:id, attr:r:id | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1432` | complexType | `CT_SlideLayoutIdList` | el:sldLayoutId | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1438` | complexType | `CT_SlideMaster` | el:cSld, group:EG_TopLevelSlide, el:sldLayoutIdLst, el:transition, el:timing, el:hf, el:txStyles, el:extLst ... | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1451` | element | `sldMaster` | - | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1452` | complexType | `CT_HandoutMaster` | el:cSld, group:EG_TopLevelSlide, el:hf, el:extLst | Out of renderer scope | Not part of current static slide-rendering goal. |
| `pml.xsd:1460` | element | `handoutMaster` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1461` | complexType | `CT_NotesMaster` | el:cSld, group:EG_TopLevelSlide, el:hf, el:notesStyle, el:extLst | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1470` | element | `notesMaster` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1471` | complexType | `CT_NotesSlide` | el:cSld, group:EG_ChildSlide, el:extLst | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1479` | element | `notes` | - | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1480` | complexType | `CT_SlideSyncProperties` | el:extLst, attr:serverSldId, attr:serverSldModifiedTime, attr:clientInsertedTime | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1488` | element | `sldSyncPr` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1489` | complexType | `CT_StringTag` | attr:name, attr:val | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1493` | complexType | `CT_TagList` | el:tag | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1498` | element | `tagLst` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1499` | simpleType | `ST_SplitterBarState` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1506` | simpleType | `ST_ViewType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1518` | complexType | `CT_NormalViewPortion` | attr:sz, attr:autoAdjust | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1522` | complexType | `CT_NormalViewProperties` | el:restoredLeft, el:restoredTop, el:extLst, attr:showOutlineIcons, attr:snapVertSplitter, attr:vertBarState, attr:horzBarState, attr:preferSingleView ... | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1534` | complexType | `CT_CommonViewProperties` | el:scale, el:origin, attr:varScale | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1541` | complexType | `CT_NotesTextViewProperties` | el:cViewPr, el:extLst | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1547` | complexType | `CT_OutlineViewSlideEntry` | attr:r:id, attr:collapse | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1551` | complexType | `CT_OutlineViewSlideList` | el:sld | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1556` | complexType | `CT_OutlineViewProperties` | el:cViewPr, el:sldLst, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1563` | complexType | `CT_SlideSorterViewProperties` | el:cViewPr, el:extLst, attr:showFormatting | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1570` | complexType | `CT_Guide` | attr:orient, attr:pos | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1574` | complexType | `CT_GuideList` | el:guide | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1579` | complexType | `CT_CommonSlideViewProperties` | el:cViewPr, el:guideLst, attr:snapToGrid, attr:snapToObjects, attr:showGuides | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1588` | complexType | `CT_SlideViewProperties` | el:cSldViewPr, el:extLst | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1594` | complexType | `CT_NotesViewProperties` | el:cSldViewPr, el:extLst | Partial | Covered by current PresentationML render/inspect support for common decks; full clause semantics remain incomplete. |
| `pml.xsd:1600` | complexType | `CT_ViewProperties` | el:normalViewPr, el:slideViewPr, el:outlineViewPr, el:notesTextViewPr, el:sorterViewPr, el:notesViewPr, el:gridSpacing, el:extLst ... | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `pml.xsd:1616` | element | `viewPr` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |

### dml-main.xsd

| Anchor | Kind | Declaration | Members | Status | Evidence / gap |
|---|---|---|---|---|---|
| `dml-main.xsd:18` | complexType | `CT_AudioFile` | el:extLst, attr:r:link, attr:contentType | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:25` | complexType | `CT_VideoFile` | el:extLst, attr:r:link, attr:contentType | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:32` | complexType | `CT_QuickTimeFile` | el:extLst, attr:r:link | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:38` | complexType | `CT_AudioCDTime` | attr:track, attr:time | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:42` | complexType | `CT_AudioCD` | el:st, el:end, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:49` | group | `EG_Media` | el:audioCd, el:wavAudioFile, el:audioFile, el:videoFile, el:quickTimeFile | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:54` | element | `videoFile` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:59` | simpleType | `ST_StyleMatrixColumnIndex` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:62` | simpleType | `ST_FontCollectionIndex` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:69` | simpleType | `ST_ColorSchemeIndex` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:85` | complexType | `CT_ColorScheme` | el:dk1, el:lt1, el:dk2, el:lt2, el:accent1, el:accent2, el:accent3, el:accent4 ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:103` | complexType | `CT_CustomColor` | group:EG_ColorChoice, attr:name | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:109` | complexType | `CT_SupplementalFont` | attr:script, attr:typeface | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:113` | complexType | `CT_CustomColorList` | el:custClr | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:118` | complexType | `CT_FontCollection` | el:latin, el:ea, el:cs, el:font, el:extLst | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:127` | complexType | `CT_EffectStyleItem` | group:EG_EffectProperties, el:scene3d, el:sp3d | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:134` | complexType | `CT_FontScheme` | el:majorFont, el:minorFont, el:extLst, attr:name | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:142` | complexType | `CT_FillStyleList` | group:EG_FillProperties | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:147` | complexType | `CT_LineStyleList` | el:ln | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:152` | complexType | `CT_EffectStyleList` | el:effectStyle | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:158` | complexType | `CT_BackgroundFillStyleList` | group:EG_FillProperties | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:163` | complexType | `CT_StyleMatrix` | el:fillStyleLst, el:lnStyleLst, el:effectStyleLst, el:bgFillStyleLst, attr:name | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:173` | complexType | `CT_BaseStyles` | el:clrScheme, el:fontScheme, el:fmtScheme, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:181` | complexType | `CT_OfficeArtExtension` | attr:uri | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:187` | simpleType | `ST_Coordinate` | - | Supported | Core unit/value type used by current render geometry or package workflows. |
| `dml-main.xsd:190` | simpleType | `ST_CoordinateUnqualified` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:196` | simpleType | `ST_Coordinate32` | - | Supported | Core unit/value type used by current render geometry or package workflows. |
| `dml-main.xsd:199` | simpleType | `ST_Coordinate32Unqualified` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:202` | simpleType | `ST_PositiveCoordinate` | - | Supported | Core unit/value type used by current render geometry or package workflows. |
| `dml-main.xsd:208` | simpleType | `ST_PositiveCoordinate32` | - | Supported | Core unit/value type used by current render geometry or package workflows. |
| `dml-main.xsd:213` | simpleType | `ST_Angle` | - | Supported | Core unit/value type used by current render geometry or package workflows. |
| `dml-main.xsd:216` | complexType | `CT_Angle` | attr:val | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:219` | simpleType | `ST_FixedAngle` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:225` | simpleType | `ST_PositiveFixedAngle` | - | Supported | Core unit/value type used by current render geometry or package workflows. |
| `dml-main.xsd:231` | complexType | `CT_PositiveFixedAngle` | attr:val | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:234` | simpleType | `ST_Percentage` | - | Supported | Core unit/value type used by current render geometry or package workflows. |
| `dml-main.xsd:237` | complexType | `CT_Percentage` | attr:val | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:240` | simpleType | `ST_PositivePercentage` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:243` | complexType | `CT_PositivePercentage` | attr:val | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:246` | simpleType | `ST_FixedPercentage` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:249` | complexType | `CT_FixedPercentage` | attr:val | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:252` | simpleType | `ST_PositiveFixedPercentage` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:255` | complexType | `CT_PositiveFixedPercentage` | attr:val | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:258` | complexType | `CT_Ratio` | attr:n, attr:d | Supported | Core unit/value type used by current render geometry or package workflows. |
| `dml-main.xsd:262` | complexType | `CT_Point2D` | attr:x, attr:y | Supported | Core unit/value type used by current render geometry or package workflows. |
| `dml-main.xsd:266` | complexType | `CT_PositiveSize2D` | attr:cx, attr:cy | Supported | Core unit/value type used by current render geometry or package workflows. |
| `dml-main.xsd:270` | complexType | `CT_ComplementTransform` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:271` | complexType | `CT_InverseTransform` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:272` | complexType | `CT_GrayscaleTransform` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:273` | complexType | `CT_GammaTransform` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:274` | complexType | `CT_InverseGammaTransform` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:275` | group | `EG_ColorTransform` | el:tint, el:shade, el:comp, el:inv, el:gray, el:alpha, el:alphaOff, el:alphaMod ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:307` | complexType | `CT_ScRgbColor` | group:EG_ColorTransform, attr:r, attr:g, attr:b | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:315` | complexType | `CT_SRgbColor` | group:EG_ColorTransform, attr:val | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:321` | complexType | `CT_HslColor` | group:EG_ColorTransform, attr:hue, attr:sat, attr:lum | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:329` | simpleType | `ST_SystemColorVal` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:363` | complexType | `CT_SystemColor` | group:EG_ColorTransform, attr:val, attr:lastClr | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:370` | simpleType | `ST_SchemeColorVal` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:391` | complexType | `CT_SchemeColor` | group:EG_ColorTransform, attr:val | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:397` | simpleType | `ST_PresetColorVal` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:591` | complexType | `CT_PresetColor` | group:EG_ColorTransform, attr:val | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:597` | group | `EG_OfficeArtExtensionList` | el:ext | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:602` | complexType | `CT_OfficeArtExtensionList` | group:EG_OfficeArtExtensionList | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:607` | complexType | `CT_Scale2D` | el:sx, el:sy | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:613` | complexType | `CT_Transform2D` | el:off, el:ext, attr:rot, attr:flipH, attr:flipV | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:622` | complexType | `CT_GroupTransform2D` | el:off, el:ext, el:chOff, el:chExt, attr:rot, attr:flipH, attr:flipV | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:633` | complexType | `CT_Point3D` | attr:x, attr:y, attr:z | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:638` | complexType | `CT_Vector3D` | attr:dx, attr:dy, attr:dz | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:643` | complexType | `CT_SphereCoords` | attr:lat, attr:lon, attr:rev | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:648` | complexType | `CT_RelativeRect` | attr:l, attr:t, attr:r, attr:b | Supported | Core unit/value type used by current render geometry or package workflows. |
| `dml-main.xsd:654` | simpleType | `ST_RectAlignment` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:105` | group | `EG_ColorChoice` | el:scrgbClr, el:srgbClr, el:hslClr, el:sysClr, el:schemeClr, el:prstClr | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:677` | complexType | `CT_Color` | group:EG_ColorChoice | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:682` | complexType | `CT_ColorMRU` | group:EG_ColorChoice | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:687` | simpleType | `ST_BlackWhiteMode` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:702` | attributeGroup | `AG_Blob` | attr:r:embed, attr:r:link | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:706` | complexType | `CT_EmbeddedWAVAudioFile` | attr:r:embed, attr:name | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:710` | complexType | `CT_Hyperlink` | el:snd, el:extLst, attr:r:id, attr:invalidUrl, attr:action, attr:tgtFrame, attr:tooltip, attr:history ... | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:724` | simpleType | `ST_DrawingElementId` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:727` | attributeGroup | `AG_Locking` | attr:noGrp, attr:noSelect, attr:noRot, attr:noChangeAspect, attr:noMove, attr:noResize, attr:noEditPoints, attr:noAdjustHandles ... | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:739` | complexType | `CT_ConnectorLocking` | el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:745` | complexType | `CT_ShapeLocking` | el:extLst, attr:noTextEdit | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:752` | complexType | `CT_PictureLocking` | el:extLst, attr:noCrop | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:759` | complexType | `CT_GroupLocking` | el:extLst, attr:noGrp, attr:noUngrp, attr:noSelect, attr:noRot, attr:noChangeAspect, attr:noMove, attr:noResize ... | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:771` | complexType | `CT_GraphicalObjectFrameLocking` | el:extLst, attr:noGrp, attr:noDrilldown, attr:noSelect, attr:noChangeAspect, attr:noMove, attr:noResize | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:782` | complexType | `CT_ContentPartLocking` | el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:788` | complexType | `CT_NonVisualDrawingProps` | el:hlinkClick, el:hlinkHover, el:extLst, attr:id, attr:name, attr:descr, attr:hidden, attr:title ... | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:800` | complexType | `CT_NonVisualDrawingShapeProps` | el:spLocks, el:extLst, attr:txBox | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:807` | complexType | `CT_NonVisualConnectorProperties` | el:cxnSpLocks, el:stCxn, el:endCxn, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:815` | complexType | `CT_NonVisualPictureProperties` | el:picLocks, el:extLst, attr:preferRelativeResize | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:822` | complexType | `CT_NonVisualGroupDrawingShapeProps` | el:grpSpLocks, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:828` | complexType | `CT_NonVisualGraphicFrameProperties` | el:graphicFrameLocks, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:835` | complexType | `CT_NonVisualContentPartProperties` | el:cpLocks, el:extLst, attr:isComment | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:842` | complexType | `CT_GraphicalObjectData` | attr:uri | Partial | Graphic payloads are recognized for tables and simple diagrams; charts/OLE/media remain unsupported. |
| `dml-main.xsd:848` | complexType | `CT_GraphicalObject` | el:graphicData | Partial | Graphic payloads are recognized for tables and simple diagrams; charts/OLE/media remain unsupported. |
| `dml-main.xsd:853` | element | `graphic` | - | Partial | Graphic payloads are recognized for tables and simple diagrams; charts/OLE/media remain unsupported. |
| `dml-main.xsd:854` | simpleType | `ST_ChartBuildStep` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:864` | simpleType | `ST_DgmBuildStep` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:870` | complexType | `CT_AnimationDgmElement` | attr:id, attr:bldStep | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:875` | complexType | `CT_AnimationChartElement` | attr:seriesIdx, attr:categoryIdx, attr:bldStep | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:880` | complexType | `CT_AnimationElementChoice` | el:dgm, el:chart | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:886` | simpleType | `ST_AnimationBuildType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:891` | simpleType | `ST_AnimationDgmOnlyBuildType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:898` | simpleType | `ST_AnimationDgmBuildType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:901` | complexType | `CT_AnimationDgmBuildProperties` | attr:bld, attr:rev | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:905` | simpleType | `ST_AnimationChartOnlyBuildType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:913` | simpleType | `ST_AnimationChartBuildType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:916` | complexType | `CT_AnimationChartBuildProperties` | attr:bld, attr:animBg | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:920` | complexType | `CT_AnimationGraphicalObjectBuildProperties` | el:bldDgm, el:bldChart | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:926` | complexType | `CT_BackgroundFormatting` | group:EG_FillProperties, group:EG_EffectProperties | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:932` | complexType | `CT_WholeE2oFormatting` | el:ln, group:EG_EffectProperties | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:938` | complexType | `CT_GvmlUseShapeRectangle` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:939` | complexType | `CT_GvmlTextShape` | el:txBody, el:useSpRect, el:xfrm, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:949` | complexType | `CT_GvmlShapeNonVisual` | el:cNvPr, el:cNvSpPr | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:955` | complexType | `CT_GvmlShape` | el:nvSpPr, el:spPr, el:txSp, el:style, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:964` | complexType | `CT_GvmlConnectorNonVisual` | el:cNvPr, el:cNvCxnSpPr | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:971` | complexType | `CT_GvmlConnector` | el:nvCxnSpPr, el:spPr, el:style, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:979` | complexType | `CT_GvmlPictureNonVisual` | el:cNvPr, el:cNvPicPr | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:986` | complexType | `CT_GvmlPicture` | el:nvPicPr, el:blipFill, el:spPr, el:style, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:995` | complexType | `CT_GvmlGraphicFrameNonVisual` | el:cNvPr, el:cNvGraphicFramePr | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1002` | complexType | `CT_GvmlGraphicalObjectFrame` | el:nvGraphicFramePr, el:graphic, el:xfrm, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1011` | complexType | `CT_GvmlGroupShapeNonVisual` | el:cNvPr, el:cNvGrpSpPr | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1018` | complexType | `CT_GvmlGroupShape` | el:nvGrpSpPr, el:grpSpPr, el:txSp, el:sp, el:cxnSp, el:pic, el:graphicFrame, el:grpSp ... | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1033` | simpleType | `ST_PresetCameraType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1099` | simpleType | `ST_FOVAngle` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1105` | complexType | `CT_Camera` | el:rot, attr:prst, attr:fov, attr:zoom | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1113` | simpleType | `ST_LightRigDirection` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1125` | simpleType | `ST_LightRigType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1156` | complexType | `CT_LightRig` | el:rot, attr:rig, attr:dir | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1163` | complexType | `CT_Scene3D` | el:camera, el:lightRig, el:backdrop, el:extLst | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1171` | complexType | `CT_Backdrop` | el:anchor, el:norm, el:up, el:extLst | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1179` | simpleType | `ST_BevelPresetType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1195` | complexType | `CT_Bevel` | attr:w, attr:h, attr:prst | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1200` | simpleType | `ST_PresetMaterialType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1219` | complexType | `CT_Shape3D` | el:bevelT, el:bevelB, el:extrusionClr, el:contourClr, el:extLst, attr:z, attr:extrusionH, attr:contourW ... | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1233` | complexType | `CT_FlatText` | attr:z | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1236` | group | `EG_Text3D` | el:sp3d, el:flatTx | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1242` | complexType | `CT_AlphaBiLevelEffect` | attr:thresh | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1245` | complexType | `CT_AlphaCeilingEffect` | - | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1246` | complexType | `CT_AlphaFloorEffect` | - | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1247` | complexType | `CT_AlphaInverseEffect` | group:EG_ColorChoice | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1252` | complexType | `CT_AlphaModulateFixedEffect` | attr:amt | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1255` | complexType | `CT_AlphaOutsetEffect` | attr:rad | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1258` | complexType | `CT_AlphaReplaceEffect` | attr:a | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1261` | complexType | `CT_BiLevelEffect` | attr:thresh | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1264` | complexType | `CT_BlurEffect` | attr:rad, attr:grow | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1268` | complexType | `CT_ColorChangeEffect` | el:clrFrom, el:clrTo, attr:useA | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1275` | complexType | `CT_ColorReplaceEffect` | group:EG_ColorChoice | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1280` | complexType | `CT_DuotoneEffect` | group:EG_ColorChoice | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1285` | complexType | `CT_GlowEffect` | group:EG_ColorChoice, attr:rad | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1291` | complexType | `CT_GrayscaleEffect` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1292` | complexType | `CT_HSLEffect` | attr:hue, attr:sat, attr:lum | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1297` | complexType | `CT_InnerShadowEffect` | group:EG_ColorChoice, attr:blurRad, attr:dist, attr:dir | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1305` | complexType | `CT_LuminanceEffect` | attr:bright, attr:contrast | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1309` | complexType | `CT_OuterShadowEffect` | group:EG_ColorChoice, attr:blurRad, attr:dist, attr:dir, attr:sx, attr:sy, attr:kx, attr:ky ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1323` | simpleType | `ST_PresetShadowVal` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1347` | complexType | `CT_PresetShadowEffect` | group:EG_ColorChoice, attr:prst, attr:dist, attr:dir | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1355` | complexType | `CT_ReflectionEffect` | attr:blurRad, attr:stA, attr:stPos, attr:endA, attr:endPos, attr:dist, attr:dir, attr:fadeDir ... | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1371` | complexType | `CT_RelativeOffsetEffect` | attr:tx, attr:ty | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1375` | complexType | `CT_SoftEdgesEffect` | attr:rad | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1378` | complexType | `CT_TintEffect` | attr:hue, attr:amt | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1382` | complexType | `CT_TransformEffect` | attr:sx, attr:sy, attr:kx, attr:ky, attr:tx, attr:ty | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1390` | complexType | `CT_NoFillProperties` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1391` | complexType | `CT_SolidColorFillProperties` | group:EG_ColorChoice | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1396` | complexType | `CT_LinearShadeProperties` | attr:ang, attr:scaled | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1400` | simpleType | `ST_PathShadeType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1407` | complexType | `CT_PathShadeProperties` | el:fillToRect, attr:path | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1413` | group | `EG_ShadeProperties` | el:lin, el:path | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1419` | simpleType | `ST_TileFlipMode` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1427` | complexType | `CT_GradientStop` | group:EG_ColorChoice, attr:pos | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1433` | complexType | `CT_GradientStopList` | el:gs | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1438` | complexType | `CT_GradientFillProperties` | el:gsLst, group:EG_ShadeProperties, el:tileRect, attr:flip, attr:rotWithShape | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1447` | complexType | `CT_TileInfoProperties` | attr:tx, attr:ty, attr:sx, attr:sy, attr:flip, attr:algn | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1455` | complexType | `CT_StretchInfoProperties` | el:fillRect | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1460` | group | `EG_FillModeProperties` | el:tile, el:stretch | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1466` | simpleType | `ST_BlipCompression` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1475` | complexType | `CT_Blip` | el:alphaBiLevel, el:alphaCeiling, el:alphaFloor, el:alphaInv, el:alphaMod, el:alphaModFix, el:alphaRepl, el:biLevel ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1502` | complexType | `CT_BlipFillProperties` | el:blip, el:srcRect, group:EG_FillModeProperties, attr:dpi, attr:rotWithShape | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1511` | simpleType | `ST_PresetPatternVal` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1569` | complexType | `CT_PatternFillProperties` | el:fgClr, el:bgClr, attr:prst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1576` | complexType | `CT_GroupFillProperties` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:144` | group | `EG_FillProperties` | el:noFill, el:solidFill, el:gradFill, el:blipFill, el:pattFill, el:grpFill | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1587` | complexType | `CT_FillProperties` | group:EG_FillProperties | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1592` | complexType | `CT_FillEffect` | group:EG_FillProperties | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1597` | simpleType | `ST_BlendMode` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1606` | complexType | `CT_FillOverlayEffect` | group:EG_FillProperties, attr:blend | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1612` | complexType | `CT_EffectReference` | attr:ref | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1615` | group | `EG_Effect` | el:cont, el:effect, el:alphaBiLevel, el:alphaCeiling, el:alphaFloor, el:alphaInv, el:alphaMod, el:alphaModFix ... | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1649` | simpleType | `ST_EffectContainerType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1655` | complexType | `CT_EffectContainer` | group:EG_Effect, attr:type, attr:name | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1660` | complexType | `CT_AlphaModulateEffect` | el:cont | Unsupported | Visible behavior is unsupported or only reported as partial in current renderer. |
| `dml-main.xsd:1665` | complexType | `CT_BlendEffect` | el:cont, attr:blend | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1671` | complexType | `CT_EffectList` | el:blur, el:fillOverlay, el:glow, el:innerShdw, el:outerShdw, el:prstShdw, el:reflection, el:softEdge ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:129` | group | `EG_EffectProperties` | el:effectLst, el:effectDag | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1689` | complexType | `CT_EffectProperties` | group:EG_EffectProperties | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1504` | element | `blip` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1695` | simpleType | `ST_ShapeType` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1886` | simpleType | `ST_TextShapeType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1931` | simpleType | `ST_GeomGuideName` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1934` | simpleType | `ST_GeomGuideFormula` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1937` | complexType | `CT_GeomGuide` | attr:name, attr:fmla | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1941` | complexType | `CT_GeomGuideList` | el:gd | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1946` | simpleType | `ST_AdjCoordinate` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1949` | simpleType | `ST_AdjAngle` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1952` | complexType | `CT_AdjPoint2D` | attr:x, attr:y | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1956` | complexType | `CT_GeomRect` | attr:l, attr:t, attr:r, attr:b | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:1962` | complexType | `CT_XYAdjustHandle` | el:pos, attr:gdRefX, attr:minX, attr:maxX, attr:gdRefY, attr:minY, attr:maxY | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1973` | complexType | `CT_PolarAdjustHandle` | el:pos, attr:gdRefR, attr:minR, attr:maxR, attr:gdRefAng, attr:minAng, attr:maxAng | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1984` | complexType | `CT_ConnectionSite` | el:pos, attr:ang | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1990` | complexType | `CT_AdjustHandleList` | el:ahXY, el:ahPolar | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:1996` | complexType | `CT_ConnectionSiteList` | el:cxn | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2001` | complexType | `CT_Connection` | attr:id, attr:idx | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2005` | complexType | `CT_Path2DMoveTo` | el:pt | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2010` | complexType | `CT_Path2DLineTo` | el:pt | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2015` | complexType | `CT_Path2DArcTo` | attr:wR, attr:hR, attr:stAng, attr:swAng | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2021` | complexType | `CT_Path2DQuadBezierTo` | el:pt | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2026` | complexType | `CT_Path2DCubicBezierTo` | el:pt | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2031` | complexType | `CT_Path2DClose` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2032` | simpleType | `ST_PathFillMode` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2042` | complexType | `CT_Path2D` | el:close, el:moveTo, el:lnTo, el:arcTo, el:quadBezTo, el:cubicBezTo, attr:w, attr:h ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2057` | complexType | `CT_Path2DList` | el:path | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2062` | complexType | `CT_PresetGeometry2D` | el:avLst, attr:prst | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2068` | complexType | `CT_PresetTextShape` | el:avLst, attr:prst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2074` | complexType | `CT_CustomGeometry2D` | el:avLst, el:gdLst, el:ahLst, el:cxnLst, el:rect, el:pathLst | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2084` | group | `EG_Geometry` | el:custGeom, el:prstGeom | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2090` | group | `EG_TextGeometry` | el:custGeom, el:prstTxWarp | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2096` | simpleType | `ST_LineEndType` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2106` | simpleType | `ST_LineEndWidth` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2113` | simpleType | `ST_LineEndLength` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2120` | complexType | `CT_LineEndProperties` | attr:type, attr:w, attr:len | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2125` | group | `EG_LineFillProperties` | el:noFill, el:solidFill, el:gradFill, el:pattFill | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2133` | complexType | `CT_LineJoinBevel` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2134` | complexType | `CT_LineJoinRound` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2135` | complexType | `CT_LineJoinMiterProperties` | attr:lim | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2138` | group | `EG_LineJoinProperties` | el:round, el:bevel, el:miter | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2145` | simpleType | `ST_PresetLineDashVal` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2160` | complexType | `CT_PresetLineDashProperties` | attr:val | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2163` | complexType | `CT_DashStop` | attr:d, attr:sp | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2167` | complexType | `CT_DashStopList` | el:ds | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2172` | group | `EG_LineDashProperties` | el:prstDash, el:custDash | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2178` | simpleType | `ST_LineCap` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2185` | simpleType | `ST_LineWidth` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2191` | simpleType | `ST_PenAlignment` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2197` | simpleType | `ST_CompoundLine` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2206` | complexType | `CT_LineProperties` | group:EG_LineFillProperties, group:EG_LineDashProperties, group:EG_LineJoinProperties, el:headEnd, el:tailEnd, el:extLst, attr:w, attr:cap ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2220` | simpleType | `ST_ShapeID` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2223` | complexType | `CT_ShapeProperties` | el:xfrm, group:EG_Geometry, group:EG_FillProperties, el:ln, group:EG_EffectProperties, el:scene3d, el:sp3d, el:extLst ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2236` | complexType | `CT_GroupShapeProperties` | el:xfrm, group:EG_FillProperties, group:EG_EffectProperties, el:scene3d, el:extLst, attr:bwMode | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2246` | complexType | `CT_StyleMatrixReference` | group:EG_ColorChoice, attr:idx | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2252` | complexType | `CT_FontReference` | group:EG_ColorChoice, attr:idx | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2258` | complexType | `CT_ShapeStyle` | el:lnRef, el:fillRef, el:effectRef, el:fontRef | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2266` | complexType | `CT_DefaultShapeDefinition` | el:spPr, el:bodyPr, el:lstStyle, el:style, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2275` | complexType | `CT_ObjectStyleDefaults` | el:spDef, el:lnDef, el:txDef, el:extLst | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2283` | complexType | `CT_EmptyElement` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2284` | complexType | `CT_ColorMapping` | el:extLst, attr:bg1, attr:tx1, attr:bg2, attr:tx2, attr:accent1, attr:accent2, attr:accent3 ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2301` | complexType | `CT_ColorMappingOverride` | el:masterClrMapping, el:overrideClrMapping | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2309` | complexType | `CT_ColorSchemeAndMapping` | el:clrScheme, el:clrMap | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2315` | complexType | `CT_ColorSchemeList` | el:extraClrScheme | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2321` | complexType | `CT_OfficeStyleSheet` | el:themeElements, el:objectDefaults, el:extraClrSchemeLst, el:custClrLst, el:extLst, attr:name | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2331` | complexType | `CT_BaseStylesOverride` | el:clrScheme, el:fontScheme, el:fmtScheme | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2338` | complexType | `CT_ClipboardStyleSheet` | el:themeElements, el:clrMap | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2344` | element | `theme` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2345` | element | `themeOverride` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2346` | element | `themeManager` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2347` | complexType | `CT_TableCellProperties` | el:lnL, el:lnR, el:lnT, el:lnB, el:lnTlToBr, el:lnBlToTr, el:cell3D, group:EG_FillProperties ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2370` | complexType | `CT_Headers` | el:header | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2375` | complexType | `CT_TableCol` | el:extLst, attr:w | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2381` | complexType | `CT_TableGrid` | el:gridCol | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2386` | complexType | `CT_TableCell` | el:txBody, el:tcPr, el:extLst, attr:rowSpan, attr:gridSpan, attr:hMerge, attr:vMerge, attr:id ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2398` | complexType | `CT_TableRow` | el:tc, el:extLst, attr:h | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2405` | complexType | `CT_TableProperties` | group:EG_FillProperties, group:EG_EffectProperties, el:tableStyle, el:tableStyleId, el:extLst, attr:rtl, attr:firstRow, attr:firstCol ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2423` | complexType | `CT_Table` | el:tblPr, el:tblGrid, el:tr | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2430` | element | `tbl` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2431` | complexType | `CT_Cell3D` | el:bevel, el:lightRig, el:extLst, attr:prstMaterial | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2440` | group | `EG_ThemeableFillStyle` | el:fill, el:fillRef | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2446` | complexType | `CT_ThemeableLineStyle` | el:ln, el:lnRef | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2452` | group | `EG_ThemeableEffectStyle` | el:effect, el:effectRef | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2458` | group | `EG_ThemeableFontStyles` | el:font, el:fontRef | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2464` | simpleType | `ST_OnOffStyleType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2471` | complexType | `CT_TableStyleTextStyle` | group:EG_ThemeableFontStyles, group:EG_ColorChoice, el:extLst, attr:b, attr:i | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2480` | complexType | `CT_TableCellBorderStyle` | el:left, el:right, el:top, el:bottom, el:insideH, el:insideV, el:tl2br, el:tr2bl ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2493` | complexType | `CT_TableBackgroundStyle` | group:EG_ThemeableFillStyle, group:EG_ThemeableEffectStyle | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2499` | complexType | `CT_TableStyleCellStyle` | el:tcBdr, group:EG_ThemeableFillStyle, el:cell3D | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2506` | complexType | `CT_TablePartStyle` | el:tcTxStyle, el:tcStyle | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2512` | complexType | `CT_TableStyle` | el:tblBg, el:wholeTbl, el:band1H, el:band2H, el:band1V, el:band2V, el:lastCol, el:firstCol ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2533` | complexType | `CT_TableStyleList` | el:tblStyle, attr:def | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2539` | element | `tblStyleLst` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2540` | complexType | `CT_TextParagraph` | el:pPr, group:EG_TextRun, el:endParaRPr | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2547` | simpleType | `ST_TextAnchoringType` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2556` | simpleType | `ST_TextVertOverflowType` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2563` | simpleType | `ST_TextHorzOverflowType` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2569` | simpleType | `ST_TextVerticalType` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2580` | simpleType | `ST_TextWrappingType` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2586` | simpleType | `ST_TextColumnCount` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2592` | complexType | `CT_TextListStyle` | el:defPPr, el:lvl1pPr, el:lvl2pPr, el:lvl3pPr, el:lvl4pPr, el:lvl5pPr, el:lvl6pPr, el:lvl7pPr ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2607` | simpleType | `ST_TextFontScalePercentOrPercentString` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2610` | complexType | `CT_TextNormalAutofit` | attr:fontScale, attr:lnSpcReduction | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2616` | complexType | `CT_TextShapeAutofit` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2617` | complexType | `CT_TextNoAutofit` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2618` | group | `EG_TextAutofit` | el:noAutofit, el:normAutofit, el:spAutoFit | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2625` | complexType | `CT_TextBodyProperties` | el:prstTxWarp, group:EG_TextAutofit, el:scene3d, group:EG_Text3D, el:extLst, attr:rot, attr:spcFirstLastPara, attr:vertOverflow ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2653` | complexType | `CT_TextBody` | el:bodyPr, el:lstStyle, el:p | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2660` | simpleType | `ST_TextBulletStartAtNum` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2666` | simpleType | `ST_TextAutonumberScheme` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2711` | complexType | `CT_TextBulletColorFollowText` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2712` | group | `EG_TextBulletColor` | el:buClrTx, el:buClr | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2718` | simpleType | `ST_TextBulletSize` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2721` | simpleType | `ST_TextBulletSizePercent` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2726` | complexType | `CT_TextBulletSizeFollowText` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2727` | complexType | `CT_TextBulletSizePercent` | attr:val | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2730` | complexType | `CT_TextBulletSizePoint` | attr:val | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2733` | group | `EG_TextBulletSize` | el:buSzTx, el:buSzPct, el:buSzPts | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2740` | complexType | `CT_TextBulletTypefaceFollowText` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2741` | group | `EG_TextBulletTypeface` | el:buFontTx, el:buFont | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2747` | complexType | `CT_TextAutonumberBullet` | attr:type, attr:startAt | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2751` | complexType | `CT_TextCharBullet` | attr:char | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2754` | complexType | `CT_TextBlipBullet` | el:blip | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2759` | complexType | `CT_TextNoBullet` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2760` | group | `EG_TextBullet` | el:buNone, el:buAutoNum, el:buChar, el:buBlip | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2768` | simpleType | `ST_TextPoint` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2771` | simpleType | `ST_TextPointUnqualified` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2777` | simpleType | `ST_TextNonNegativePoint` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2783` | simpleType | `ST_TextFontSize` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2789` | simpleType | `ST_TextTypeface` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2792` | simpleType | `ST_PitchFamily` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2814` | complexType | `CT_TextFont` | attr:typeface, attr:panose, attr:pitchFamily, attr:charset | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2820` | simpleType | `ST_TextUnderlineType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2842` | complexType | `CT_TextUnderlineLineFollowText` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2843` | complexType | `CT_TextUnderlineFillFollowText` | - | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2844` | complexType | `CT_TextUnderlineFillGroupWrapper` | group:EG_FillProperties | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2847` | group | `EG_TextUnderlineLine` | el:uLnTx, el:uLn | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2853` | group | `EG_TextUnderlineFill` | el:uFillTx, el:uFill | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2859` | simpleType | `ST_TextStrikeType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2866` | simpleType | `ST_TextCapsType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2873` | complexType | `CT_TextCharacterProperties` | el:ln, group:EG_FillProperties, group:EG_EffectProperties, el:highlight, group:EG_TextUnderlineLine, group:EG_TextUnderlineFill, el:latin, el:ea ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2910` | complexType | `CT_Boolean` | attr:val | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2913` | simpleType | `ST_TextSpacingPoint` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2919` | simpleType | `ST_TextSpacingPercentOrPercentString` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2922` | complexType | `CT_TextSpacingPercent` | attr:val | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2925` | complexType | `CT_TextSpacingPoint` | attr:val | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2928` | simpleType | `ST_TextMargin` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2934` | simpleType | `ST_TextIndent` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2940` | simpleType | `ST_TextTabAlignType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2948` | complexType | `CT_TextTabStop` | attr:pos, attr:algn | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2952` | complexType | `CT_TextTabStopList` | el:tab | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2957` | complexType | `CT_TextLineBreak` | el:rPr | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2962` | complexType | `CT_TextSpacing` | el:spcPct, el:spcPts | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2968` | simpleType | `ST_TextAlignType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2979` | simpleType | `ST_TextFontAlignType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2988` | simpleType | `ST_TextIndentLevelType` | - | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |
| `dml-main.xsd:2994` | complexType | `CT_TextParagraphProperties` | el:lnSpc, el:spcBef, el:spcAft, group:EG_TextBulletColor, group:EG_TextBulletSize, group:EG_TextBulletTypeface, group:EG_TextBullet, el:tabLst ... | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:3019` | complexType | `CT_TextField` | el:rPr, el:pPr, el:t, attr:id, attr:type | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:2543` | group | `EG_TextRun` | el:r, el:br, el:fld | Partial | Part of current DrawingML render subset; full source semantics are not complete. |
| `dml-main.xsd:3035` | complexType | `CT_RegularTextRun` | el:rPr, el:t | Unimplemented / no evidence | No explicit renderer coverage evidence found in the maintained docs/tests. |

### dml-picture.xsd

| Anchor | Kind | Declaration | Members | Status | Evidence / gap |
|---|---|---|---|---|---|
| `dml-picture.xsd:7` | complexType | `CT_PictureNonVisual` | el:cNvPr, el:cNvPicPr | Partial | Picture object structure is parsed/rendered through PresentationML picture handling; full blip/effect behavior is incomplete. |
| `dml-picture.xsd:14` | complexType | `CT_Picture` | el:nvPicPr, el:blipFill, el:spPr | Partial | Picture object structure is parsed/rendered through PresentationML picture handling; full blip/effect behavior is incomplete. |
| `dml-picture.xsd:21` | element | `pic` | - | Partial | Picture object structure is parsed/rendered through PresentationML picture handling; full blip/effect behavior is incomplete. |

### dml-diagram.xsd

| Anchor | Kind | Declaration | Members | Status | Evidence / gap |
|---|---|---|---|---|---|
| `dml-diagram.xsd:14` | complexType | `CT_CTName` | attr:lang, attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:18` | complexType | `CT_CTDescription` | attr:lang, attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:22` | complexType | `CT_CTCategory` | attr:type, attr:pri | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:26` | complexType | `CT_CTCategories` | el:cat | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:31` | simpleType | `ST_ClrAppMethod` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:38` | simpleType | `ST_HueDir` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:44` | complexType | `CT_Colors` | group:a:EG_ColorChoice, attr:meth, attr:hueDir | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:51` | complexType | `CT_CTStyleLabel` | el:fillClrLst, el:linClrLst, el:effectClrLst, el:txLinClrLst, el:txFillClrLst, el:txEffectClrLst, el:extLst, attr:name ... | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:63` | complexType | `CT_ColorTransform` | el:title, el:desc, el:catLst, el:styleLbl, el:extLst, attr:uniqueId, attr:minVer | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:74` | element | `colorsDef` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:75` | complexType | `CT_ColorTransformHeader` | el:title, el:desc, el:catLst, el:extLst, attr:uniqueId, attr:minVer, attr:resId | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:86` | element | `colorsDefHdr` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:87` | complexType | `CT_ColorTransformHeaderLst` | el:colorsDefHdr | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:93` | element | `colorsDefHdrLst` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:94` | simpleType | `ST_PtType` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:104` | complexType | `CT_Pt` | el:prSet, el:spPr, el:t, el:extLst, attr:modelId, attr:type, attr:cxnId | Partial | Simple diagram shapes/text can render when resolved from related diagram drawing parts. |
| `dml-diagram.xsd:115` | complexType | `CT_PtList` | el:pt | Partial | Simple diagram shapes/text can render when resolved from related diagram drawing parts. |
| `dml-diagram.xsd:120` | simpleType | `ST_CxnType` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:128` | complexType | `CT_Cxn` | el:extLst, attr:modelId, attr:type, attr:srcId, attr:destId, attr:srcOrd, attr:destOrd, attr:parTransId ... | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:142` | complexType | `CT_CxnList` | el:cxn | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:147` | complexType | `CT_DataModel` | el:ptLst, el:cxnLst, el:bg, el:whole, el:extLst | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:156` | element | `dataModel` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:157` | attributeGroup | `AG_IteratorAttributes` | attr:axis, attr:ptType, attr:hideLastTrans, attr:st, attr:cnt, attr:step | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:165` | attributeGroup | `AG_ConstraintAttributes` | attr:type, attr:for, attr:forName, attr:ptType | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:171` | attributeGroup | `AG_ConstraintRefAttributes` | attr:refType, attr:refFor, attr:refForName, attr:refPtType | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:177` | complexType | `CT_Constraint` | el:extLst, attr:op, attr:val, attr:fact | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:187` | complexType | `CT_Constraints` | el:constr | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:192` | complexType | `CT_NumericRule` | el:extLst, attr:val, attr:fact, attr:max | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:201` | complexType | `CT_Rules` | el:rule | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:206` | complexType | `CT_PresentationOf` | el:extLst | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:212` | simpleType | `ST_LayoutShapeType` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:215` | simpleType | `ST_Index1` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:220` | complexType | `CT_Adj` | attr:idx, attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:224` | complexType | `CT_AdjLst` | el:adj | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:229` | complexType | `CT_Shape` | el:adjLst, el:extLst, attr:rot, attr:type, attr:r:blip, attr:zOrderOff, attr:hideGeom, attr:lkTxEntry ... | Partial | Simple diagram shapes/text can render when resolved from related diagram drawing parts. |
| `dml-diagram.xsd:242` | complexType | `CT_Parameter` | attr:type, attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:246` | complexType | `CT_Algorithm` | el:param, el:extLst, attr:type, attr:rev | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:254` | complexType | `CT_LayoutNode` | el:alg, el:shape, el:presOf, el:constrLst, el:ruleLst, el:varLst, el:forEach, el:layoutNode ... | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:272` | complexType | `CT_ForEach` | el:alg, el:shape, el:presOf, el:constrLst, el:ruleLst, el:forEach, el:layoutNode, el:choose ... | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:288` | complexType | `CT_When` | el:alg, el:shape, el:presOf, el:constrLst, el:ruleLst, el:forEach, el:layoutNode, el:choose ... | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:307` | complexType | `CT_Otherwise` | el:alg, el:shape, el:presOf, el:constrLst, el:ruleLst, el:forEach, el:layoutNode, el:choose ... | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:321` | complexType | `CT_Choose` | el:if, el:else, attr:name | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:328` | complexType | `CT_SampleData` | el:dataModel, attr:useDef | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:334` | complexType | `CT_Category` | attr:type, attr:pri | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:338` | complexType | `CT_Categories` | el:cat | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:343` | complexType | `CT_Name` | attr:lang, attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:347` | complexType | `CT_Description` | attr:lang, attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:351` | complexType | `CT_DiagramDefinition` | el:title, el:desc, el:catLst, el:sampData, el:styleData, el:clrData, el:layoutNode, el:extLst ... | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:366` | element | `layoutDef` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:367` | complexType | `CT_DiagramDefinitionHeader` | el:title, el:desc, el:catLst, el:extLst, attr:uniqueId, attr:minVer, attr:defStyle, attr:resId ... | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:379` | element | `layoutDefHdr` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:380` | complexType | `CT_DiagramDefinitionHeaderLst` | el:layoutDefHdr | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:386` | element | `layoutDefHdrLst` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:387` | complexType | `CT_RelIds` | attr:r:dm, attr:r:lo, attr:r:qs, attr:r:cs | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:393` | element | `relIds` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:394` | simpleType | `ST_ParameterVal` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:399` | simpleType | `ST_ModelId` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:402` | simpleType | `ST_PrSetCustVal` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:405` | complexType | `CT_ElemPropSet` | el:presLayoutVars, el:style, attr:presAssocID, attr:presName, attr:presStyleLbl, attr:presStyleIdx, attr:presStyleCnt, attr:loTypeId ... | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:440` | simpleType | `ST_Direction` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:446` | simpleType | `ST_HierBranchStyle` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:455` | simpleType | `ST_AnimOneStr` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:462` | simpleType | `ST_AnimLvlStr` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:469` | complexType | `CT_OrgChart` | attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:472` | simpleType | `ST_NodeCount` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:477` | complexType | `CT_ChildMax` | attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:480` | complexType | `CT_ChildPref` | attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:483` | complexType | `CT_BulletEnabled` | attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:486` | complexType | `CT_Direction` | attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:489` | complexType | `CT_HierBranchStyle` | attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:492` | complexType | `CT_AnimOne` | attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:495` | complexType | `CT_AnimLvl` | attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:498` | simpleType | `ST_ResizeHandlesStr` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:504` | complexType | `CT_ResizeHandles` | attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:507` | complexType | `CT_LayoutVariablePropertySet` | el:orgChart, el:chMax, el:chPref, el:bulletEnabled, el:dir, el:hierBranch, el:animOne, el:animLvl ... | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:520` | complexType | `CT_SDName` | attr:lang, attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:524` | complexType | `CT_SDDescription` | attr:lang, attr:val | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:528` | complexType | `CT_SDCategory` | attr:type, attr:pri | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:532` | complexType | `CT_SDCategories` | el:cat | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:537` | complexType | `CT_TextProps` | group:a:EG_Text3D | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:542` | complexType | `CT_StyleLabel` | el:scene3d, el:sp3d, el:txPr, el:style, el:extLst, attr:name | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:552` | complexType | `CT_StyleDefinition` | el:title, el:desc, el:catLst, el:scene3d, el:styleLbl, el:extLst, attr:uniqueId, attr:minVer ... | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:564` | element | `styleDef` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:565` | complexType | `CT_StyleDefinitionHeader` | el:title, el:desc, el:catLst, el:extLst, attr:uniqueId, attr:minVer, attr:resId | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:576` | element | `styleDefHdr` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:577` | complexType | `CT_StyleDefinitionHeaderLst` | el:styleDefHdr | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:583` | element | `styleDefHdrLst` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:584` | simpleType | `ST_AlgorithmType` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:598` | simpleType | `ST_AxisType` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:615` | simpleType | `ST_AxisTypes` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:618` | simpleType | `ST_BoolOperator` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:626` | simpleType | `ST_ChildOrderType` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:632` | simpleType | `ST_ConstraintType` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:700` | simpleType | `ST_ConstraintRelationship` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:707` | simpleType | `ST_ElementType` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:721` | simpleType | `ST_ElementTypes` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:724` | simpleType | `ST_ParameterId` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:783` | simpleType | `ST_Ints` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:786` | simpleType | `ST_UnsignedInts` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:789` | simpleType | `ST_Booleans` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:792` | simpleType | `ST_FunctionType` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:804` | simpleType | `ST_FunctionOperator` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:814` | simpleType | `ST_DiagramHorizontalAlignment` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:822` | simpleType | `ST_VerticalAlignment` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:830` | simpleType | `ST_ChildDirection` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:836` | simpleType | `ST_ChildAlignment` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:844` | simpleType | `ST_SecondaryChildAlignment` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:853` | simpleType | `ST_LinearDirection` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:861` | simpleType | `ST_SecondaryLinearDirection` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:870` | simpleType | `ST_StartingElement` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:876` | simpleType | `ST_RotationPath` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:882` | simpleType | `ST_CenterShapeMapping` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:888` | simpleType | `ST_BendPoint` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:895` | simpleType | `ST_ConnectorRouting` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:903` | simpleType | `ST_ArrowheadStyle` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:910` | simpleType | `ST_ConnectorDimension` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:917` | simpleType | `ST_ConnectorPoint` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:932` | simpleType | `ST_NodeHorizontalAlignment` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:939` | simpleType | `ST_NodeVerticalAlignment` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:946` | simpleType | `ST_FallbackDimension` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:952` | simpleType | `ST_TextDirection` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:958` | simpleType | `ST_PyramidAccentPosition` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:964` | simpleType | `ST_PyramidAccentTextMargin` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:970` | simpleType | `ST_TextBlockDirection` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:976` | simpleType | `ST_TextAnchorHorizontal` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:982` | simpleType | `ST_TextAnchorVertical` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:989` | simpleType | `ST_DiagramTextAlignment` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:996` | simpleType | `ST_AutoTextRotation` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:1003` | simpleType | `ST_GrowDirection` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:1011` | simpleType | `ST_FlowDirection` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:1017` | simpleType | `ST_ContinueDirection` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:1023` | simpleType | `ST_Breakpoint` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:1030` | simpleType | `ST_Offset` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:1036` | simpleType | `ST_HierarchyAlignment` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:1056` | simpleType | `ST_FunctionValue` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:1061` | simpleType | `ST_VariableType` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:1075` | simpleType | `ST_FunctionArgument` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |
| `dml-diagram.xsd:1078` | simpleType | `ST_OutputShapeType` | - | Unsupported | SmartArt/diagram layout and non-shape content are not fully implemented. |

### dml-chart.xsd

| Anchor | Kind | Declaration | Members | Status | Evidence / gap |
|---|---|---|---|---|---|
| `dml-chart.xsd:17` | complexType | `CT_Boolean` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:20` | complexType | `CT_Double` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:23` | complexType | `CT_UnsignedInt` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:26` | complexType | `CT_RelId` | attr:r:id | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:29` | complexType | `CT_Extension` | attr:uri | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:35` | complexType | `CT_ExtensionList` | el:ext | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:40` | complexType | `CT_NumVal` | el:v, attr:idx, attr:formatCode | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:47` | complexType | `CT_NumData` | el:formatCode, el:ptCount, el:pt, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:55` | complexType | `CT_NumRef` | el:f, el:numCache, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:62` | complexType | `CT_NumDataSource` | el:numRef, el:numLit | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:70` | complexType | `CT_StrVal` | el:v, attr:idx | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:76` | complexType | `CT_StrData` | el:ptCount, el:pt, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:83` | complexType | `CT_StrRef` | el:f, el:strCache, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:90` | complexType | `CT_Tx` | el:strRef, el:rich | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:98` | complexType | `CT_TextLanguageID` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:101` | complexType | `CT_Lvl` | el:pt | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:106` | complexType | `CT_MultiLvlStrData` | el:ptCount, el:lvl, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:113` | complexType | `CT_MultiLvlStrRef` | el:f, el:multiLvlStrCache, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:120` | complexType | `CT_AxDataSource` | el:multiLvlStrRef, el:numRef, el:numLit, el:strRef, el:strLit | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:131` | complexType | `CT_SerTx` | el:strRef, el:v | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:139` | simpleType | `ST_LayoutTarget` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:145` | complexType | `CT_LayoutTarget` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:148` | simpleType | `ST_LayoutMode` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:154` | complexType | `CT_LayoutMode` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:157` | complexType | `CT_ManualLayout` | el:layoutTarget, el:xMode, el:yMode, el:wMode, el:hMode, el:x, el:y, el:w ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:171` | complexType | `CT_Layout` | el:manualLayout, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:177` | complexType | `CT_Title` | el:tx, el:layout, el:overlay, el:spPr, el:txPr, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:187` | simpleType | `ST_RotX` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:193` | complexType | `CT_RotX` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:196` | simpleType | `ST_HPercent` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:199` | simpleType | `ST_HPercentWithSymbol` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:204` | complexType | `CT_HPercent` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:207` | simpleType | `ST_RotY` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:213` | complexType | `CT_RotY` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:216` | simpleType | `ST_DepthPercent` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:219` | simpleType | `ST_DepthPercentWithSymbol` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:224` | complexType | `CT_DepthPercent` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:227` | simpleType | `ST_Perspective` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:233` | complexType | `CT_Perspective` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:236` | complexType | `CT_View3D` | el:rotX, el:hPercent, el:rotY, el:depthPercent, el:rAngAx, el:perspective, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:247` | complexType | `CT_Surface` | el:thickness, el:spPr, el:pictureOptions, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:255` | simpleType | `ST_Thickness` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:258` | simpleType | `ST_ThicknessPercent` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:263` | complexType | `CT_Thickness` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:266` | complexType | `CT_DTable` | el:showHorzBorder, el:showVertBorder, el:showOutline, el:showKeys, el:spPr, el:txPr, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:277` | simpleType | `ST_GapAmount` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:280` | simpleType | `ST_GapAmountPercent` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:285` | complexType | `CT_GapAmount` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:288` | simpleType | `ST_Overlap` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:291` | simpleType | `ST_OverlapPercent` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:296` | complexType | `CT_Overlap` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:299` | simpleType | `ST_BubbleScale` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:302` | simpleType | `ST_BubbleScalePercent` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:307` | complexType | `CT_BubbleScale` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:310` | simpleType | `ST_SizeRepresents` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:316` | complexType | `CT_SizeRepresents` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:319` | simpleType | `ST_FirstSliceAng` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:325` | complexType | `CT_FirstSliceAng` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:328` | simpleType | `ST_HoleSize` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:331` | simpleType | `ST_HoleSizePercent` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:336` | complexType | `CT_HoleSize` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:339` | simpleType | `ST_SplitType` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:348` | complexType | `CT_SplitType` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:351` | complexType | `CT_CustSplit` | el:secondPiePt | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:356` | simpleType | `ST_SecondPieSize` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:359` | simpleType | `ST_SecondPieSizePercent` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:364` | complexType | `CT_SecondPieSize` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:367` | complexType | `CT_NumFmt` | attr:formatCode, attr:sourceLinked | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:371` | simpleType | `ST_LblAlgn` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:378` | complexType | `CT_LblAlgn` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:381` | simpleType | `ST_DLblPos` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:394` | complexType | `CT_DLblPos` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:397` | group | `EG_DLblShared` | el:numFmt, el:spPr, el:txPr, el:dLblPos, el:showLegendKey, el:showVal, el:showCatName, el:showSerName ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:412` | group | `Group_DLbl` | el:layout, el:tx, group:EG_DLblShared | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:419` | complexType | `CT_DLbl` | el:idx, el:delete, group:Group_DLbl, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:429` | group | `Group_DLbls` | group:EG_DLblShared, el:showLeaderLines, el:leaderLines | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:436` | complexType | `CT_DLbls` | el:dLbl, el:delete, group:Group_DLbls, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:446` | simpleType | `ST_MarkerStyle` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:462` | complexType | `CT_MarkerStyle` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:465` | simpleType | `ST_MarkerSize` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:471` | complexType | `CT_MarkerSize` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:474` | complexType | `CT_Marker` | el:symbol, el:size, el:spPr, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:482` | complexType | `CT_DPt` | el:idx, el:invertIfNegative, el:marker, el:bubble3D, el:explosion, el:spPr, el:pictureOptions, el:extLst ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:494` | simpleType | `ST_TrendlineType` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:504` | complexType | `CT_TrendlineType` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:507` | simpleType | `ST_Order` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:513` | complexType | `CT_Order` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:516` | simpleType | `ST_Period` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:521` | complexType | `CT_Period` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:524` | complexType | `CT_TrendlineLbl` | el:layout, el:tx, el:numFmt, el:spPr, el:txPr, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:534` | complexType | `CT_Trendline` | el:name, el:spPr, el:trendlineType, el:order, el:period, el:forward, el:backward, el:intercept ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:550` | simpleType | `ST_ErrDir` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:556` | complexType | `CT_ErrDir` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:559` | simpleType | `ST_ErrBarType` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:566` | complexType | `CT_ErrBarType` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:569` | simpleType | `ST_ErrValType` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:578` | complexType | `CT_ErrValType` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:581` | complexType | `CT_ErrBars` | el:errDir, el:errBarType, el:errValType, el:noEndCap, el:plus, el:minus, el:val, el:spPr ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:594` | complexType | `CT_UpDownBar` | el:spPr | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:599` | complexType | `CT_UpDownBars` | el:gapWidth, el:upBars, el:downBars, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:607` | group | `EG_SerShared` | el:idx, el:order, el:tx, el:spPr | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:615` | complexType | `CT_LineSer` | group:EG_SerShared, el:marker, el:dPt, el:dLbls, el:trendline, el:errBars, el:cat, el:val ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:629` | complexType | `CT_ScatterSer` | group:EG_SerShared, el:marker, el:dPt, el:dLbls, el:trendline, el:errBars, el:xVal, el:yVal ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:643` | complexType | `CT_RadarSer` | group:EG_SerShared, el:marker, el:dPt, el:dLbls, el:cat, el:val, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:654` | complexType | `CT_BarSer` | group:EG_SerShared, el:invertIfNegative, el:pictureOptions, el:dPt, el:dLbls, el:trendline, el:errBars, el:cat ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:669` | complexType | `CT_AreaSer` | group:EG_SerShared, el:pictureOptions, el:dPt, el:dLbls, el:trendline, el:errBars, el:cat, el:val ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:682` | complexType | `CT_PieSer` | group:EG_SerShared, el:explosion, el:dPt, el:dLbls, el:cat, el:val, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:693` | complexType | `CT_BubbleSer` | group:EG_SerShared, el:invertIfNegative, el:dPt, el:dLbls, el:trendline, el:errBars, el:xVal, el:yVal ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:708` | complexType | `CT_SurfaceSer` | group:EG_SerShared, el:cat, el:val, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:716` | simpleType | `ST_Grouping` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:723` | complexType | `CT_Grouping` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:726` | complexType | `CT_ChartLines` | el:spPr | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:731` | group | `EG_LineChartShared` | el:grouping, el:varyColors, el:ser, el:dLbls, el:dropLines | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:740` | complexType | `CT_LineChart` | group:EG_LineChartShared, el:hiLowLines, el:upDownBars, el:marker, el:smooth, el:axId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:751` | complexType | `CT_Line3DChart` | group:EG_LineChartShared, el:gapDepth, el:axId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:759` | complexType | `CT_StockChart` | el:ser, el:dLbls, el:dropLines, el:hiLowLines, el:upDownBars, el:axId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:770` | simpleType | `ST_ScatterStyle` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:780` | complexType | `CT_ScatterStyle` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:783` | complexType | `CT_ScatterChart` | el:scatterStyle, el:varyColors, el:ser, el:dLbls, el:axId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:793` | simpleType | `ST_RadarStyle` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:800` | complexType | `CT_RadarStyle` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:803` | complexType | `CT_RadarChart` | el:radarStyle, el:varyColors, el:ser, el:dLbls, el:axId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:813` | simpleType | `ST_BarGrouping` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:821` | complexType | `CT_BarGrouping` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:824` | simpleType | `ST_BarDir` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:830` | complexType | `CT_BarDir` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:833` | simpleType | `ST_Shape` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:843` | complexType | `CT_Shape` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:846` | group | `EG_BarChartShared` | el:barDir, el:grouping, el:varyColors, el:ser, el:dLbls | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:855` | complexType | `CT_BarChart` | group:EG_BarChartShared, el:gapWidth, el:overlap, el:serLines, el:axId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:865` | complexType | `CT_Bar3DChart` | group:EG_BarChartShared, el:gapWidth, el:gapDepth, el:shape, el:axId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:875` | group | `EG_AreaChartShared` | el:grouping, el:varyColors, el:ser, el:dLbls, el:dropLines | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:884` | complexType | `CT_AreaChart` | group:EG_AreaChartShared, el:axId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:891` | complexType | `CT_Area3DChart` | group:EG_AreaChartShared, el:gapDepth, el:axId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:899` | group | `EG_PieChartShared` | el:varyColors, el:ser, el:dLbls | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:906` | complexType | `CT_PieChart` | group:EG_PieChartShared, el:firstSliceAng, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:913` | complexType | `CT_Pie3DChart` | group:EG_PieChartShared, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:919` | complexType | `CT_DoughnutChart` | group:EG_PieChartShared, el:firstSliceAng, el:holeSize, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:927` | simpleType | `ST_OfPieType` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:933` | complexType | `CT_OfPieType` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:936` | complexType | `CT_OfPieChart` | el:ofPieType, group:EG_PieChartShared, el:gapWidth, el:splitType, el:splitPos, el:custSplit, el:secondPieSize, el:serLines ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:949` | complexType | `CT_BubbleChart` | el:varyColors, el:ser, el:dLbls, el:bubble3D, el:bubbleScale, el:showNegBubbles, el:sizeRepresents, el:axId ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:962` | complexType | `CT_BandFmt` | el:idx, el:spPr | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:968` | complexType | `CT_BandFmts` | el:bandFmt | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:973` | group | `EG_SurfaceChartShared` | el:wireframe, el:ser, el:bandFmts | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:980` | complexType | `CT_SurfaceChart` | group:EG_SurfaceChartShared, el:axId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:987` | complexType | `CT_Surface3DChart` | group:EG_SurfaceChartShared, el:axId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:994` | simpleType | `ST_AxPos` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1002` | complexType | `CT_AxPos` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1005` | simpleType | `ST_Crosses` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1012` | complexType | `CT_Crosses` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1015` | simpleType | `ST_CrossBetween` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1021` | complexType | `CT_CrossBetween` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1024` | simpleType | `ST_TickMark` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1032` | complexType | `CT_TickMark` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1035` | simpleType | `ST_TickLblPos` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1043` | complexType | `CT_TickLblPos` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1046` | simpleType | `ST_Skip` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1051` | complexType | `CT_Skip` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1054` | simpleType | `ST_TimeUnit` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1061` | complexType | `CT_TimeUnit` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1064` | simpleType | `ST_AxisUnit` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1069` | complexType | `CT_AxisUnit` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1072` | simpleType | `ST_BuiltInUnit` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1085` | complexType | `CT_BuiltInUnit` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1088` | simpleType | `ST_PictureFormat` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1095` | complexType | `CT_PictureFormat` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1098` | simpleType | `ST_PictureStackUnit` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1103` | complexType | `CT_PictureStackUnit` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1106` | complexType | `CT_PictureOptions` | el:applyToFront, el:applyToSides, el:applyToEnd, el:pictureFormat, el:pictureStackUnit | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1115` | complexType | `CT_DispUnitsLbl` | el:layout, el:tx, el:spPr, el:txPr | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1123` | complexType | `CT_DispUnits` | el:custUnit, el:builtInUnit, el:dispUnitsLbl, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1133` | simpleType | `ST_Orientation` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1139` | complexType | `CT_Orientation` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1142` | simpleType | `ST_LogBase` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1148` | complexType | `CT_LogBase` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1151` | complexType | `CT_Scaling` | el:logBase, el:orientation, el:max, el:min, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1160` | simpleType | `ST_LblOffset` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1163` | simpleType | `ST_LblOffsetPercent` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1168` | complexType | `CT_LblOffset` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1171` | group | `EG_AxShared` | el:axId, el:scaling, el:delete, el:axPos, el:majorGridlines, el:minorGridlines, el:title, el:numFmt ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1193` | complexType | `CT_CatAx` | group:EG_AxShared, el:auto, el:lblAlgn, el:lblOffset, el:tickLblSkip, el:tickMarkSkip, el:noMultiLvlLbl, el:extLst ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1205` | complexType | `CT_DateAx` | group:EG_AxShared, el:auto, el:lblOffset, el:baseTimeUnit, el:majorUnit, el:majorTimeUnit, el:minorUnit, el:minorTimeUnit ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1218` | complexType | `CT_SerAx` | group:EG_AxShared, el:tickLblSkip, el:tickMarkSkip, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1226` | complexType | `CT_ValAx` | group:EG_AxShared, el:crossBetween, el:majorUnit, el:minorUnit, el:dispUnits, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1236` | complexType | `CT_PlotArea` | el:layout, el:areaChart, el:area3DChart, el:lineChart, el:line3DChart, el:stockChart, el:radarChart, el:scatterChart ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1268` | complexType | `CT_PivotFmt` | el:idx, el:spPr, el:txPr, el:marker, el:dLbl, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1278` | complexType | `CT_PivotFmts` | el:pivotFmt | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1283` | simpleType | `ST_LegendPos` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1292` | complexType | `CT_LegendPos` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1295` | group | `EG_LegendEntryData` | el:txPr | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1300` | complexType | `CT_LegendEntry` | el:idx, el:delete, group:EG_LegendEntryData, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1310` | complexType | `CT_Legend` | el:legendPos, el:legendEntry, el:layout, el:overlay, el:spPr, el:txPr, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1321` | simpleType | `ST_DispBlanksAs` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1328` | complexType | `CT_DispBlanksAs` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1331` | complexType | `CT_Chart` | el:title, el:autoTitleDeleted, el:pivotFmts, el:view3D, el:floor, el:sideWall, el:backWall, el:plotArea ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1348` | simpleType | `ST_Style` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1354` | complexType | `CT_Style` | attr:val | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1357` | complexType | `CT_PivotSource` | el:name, el:fmtId, el:extLst | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1364` | complexType | `CT_Protection` | el:chartObject, el:data, el:formatting, el:selection, el:userInterface | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1373` | complexType | `CT_HeaderFooter` | el:oddHeader, el:oddFooter, el:evenHeader, el:evenFooter, el:firstHeader, el:firstFooter, attr:alignWithMargins, attr:differentOddEven ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1386` | complexType | `CT_PageMargins` | attr:l, attr:r, attr:t, attr:b, attr:header, attr:footer | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1394` | simpleType | `ST_PageSetupOrientation` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1401` | complexType | `CT_ExternalData` | el:autoUpdate, attr:r:id | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1407` | complexType | `CT_PageSetup` | attr:paperSize, attr:paperHeight, attr:paperWidth, attr:firstPageNumber, attr:orientation, attr:blackAndWhite, attr:draft, attr:useFirstPageNumber ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1421` | complexType | `CT_PrintSettings` | el:headerFooter, el:pageMargins, el:pageSetup | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1428` | complexType | `CT_ChartSpace` | el:date1904, el:lang, el:roundedCorners, el:style, el:clrMapOvr, el:pivotSource, el:protection, el:chart ... | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1446` | element | `chartSpace` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1442` | element | `userShapes` | - | Unsupported | Chart schema is not rendered as chart graphics. |
| `dml-chart.xsd:1437` | element | `chart` | - | Unsupported | Chart schema is not rendered as chart graphics. |

### dml-chartDrawing.xsd

| Anchor | Kind | Declaration | Members | Status | Evidence / gap |
|---|---|---|---|---|---|
| `dml-chartDrawing.xsd:7` | complexType | `CT_ShapeNonVisual` | el:cNvPr, el:cNvSpPr | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:14` | complexType | `CT_Shape` | el:nvSpPr, el:spPr, el:style, el:txBody, attr:macro, attr:textlink, attr:fLocksText, attr:fPublished ... | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:26` | complexType | `CT_ConnectorNonVisual` | el:cNvPr, el:cNvCxnSpPr | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:33` | complexType | `CT_Connector` | el:nvCxnSpPr, el:spPr, el:style, attr:macro, attr:fPublished | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:42` | complexType | `CT_PictureNonVisual` | el:cNvPr, el:cNvPicPr | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:49` | complexType | `CT_Picture` | el:nvPicPr, el:blipFill, el:spPr, el:style, attr:macro, attr:fPublished | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:59` | complexType | `CT_GraphicFrameNonVisual` | el:cNvPr, el:cNvGraphicFramePr | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:66` | complexType | `CT_GraphicFrame` | el:nvGraphicFramePr, el:xfrm, el:a:graphic, attr:macro, attr:fPublished | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:76` | complexType | `CT_GroupShapeNonVisual` | el:cNvPr, el:cNvGrpSpPr | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:83` | complexType | `CT_GroupShape` | el:nvGrpSpPr, el:grpSpPr, el:sp, el:grpSp, el:graphicFrame, el:cxnSp, el:pic | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:96` | group | `EG_ObjectChoices` | el:sp, el:grpSp, el:graphicFrame, el:cxnSp, el:pic | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:107` | simpleType | `ST_MarkerCoordinate` | - | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:113` | complexType | `CT_Marker` | el:x, el:y | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:119` | complexType | `CT_RelSizeAnchor` | el:from, el:to, group:EG_ObjectChoices | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:126` | complexType | `CT_AbsSizeAnchor` | el:from, el:ext, group:EG_ObjectChoices | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:133` | group | `EG_Anchor` | el:relSizeAnchor, el:absSizeAnchor | Unsupported | Chart drawing schema is not rendered directly. |
| `dml-chartDrawing.xsd:139` | complexType | `CT_Drawing` | group:EG_Anchor | Unsupported | Chart drawing schema is not rendered directly. |

### dml-lockedCanvas.xsd

| Anchor | Kind | Declaration | Members | Status | Evidence / gap |
|---|---|---|---|---|---|
| `dml-lockedCanvas.xsd:8` | element | `lockedCanvas` | - | Unsupported | Locked canvas is not lowered into render primitives. |

### dml-spreadsheetDrawing.xsd

| Anchor | Kind | Declaration | Members | Status | Evidence / gap |
|---|---|---|---|---|---|
| `dml-spreadsheetDrawing.xsd:11` | element | `from` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:12` | element | `to` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:13` | complexType | `CT_AnchorClientData` | attr:fLocksWithSheet, attr:fPrintsWithSheet | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:17` | complexType | `CT_ShapeNonVisual` | el:cNvPr, el:cNvSpPr | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:24` | complexType | `CT_Shape` | el:nvSpPr, el:spPr, el:style, el:txBody, attr:macro, attr:textlink, attr:fLocksText, attr:fPublished ... | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:36` | complexType | `CT_ConnectorNonVisual` | el:cNvPr, el:cNvCxnSpPr | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:43` | complexType | `CT_Connector` | el:nvCxnSpPr, el:spPr, el:style, attr:macro, attr:fPublished | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:52` | complexType | `CT_PictureNonVisual` | el:cNvPr, el:cNvPicPr | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:59` | complexType | `CT_Picture` | el:nvPicPr, el:blipFill, el:spPr, el:style, attr:macro, attr:fPublished | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:69` | complexType | `CT_GraphicalObjectFrameNonVisual` | el:cNvPr, el:cNvGraphicFramePr | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:76` | complexType | `CT_GraphicalObjectFrame` | el:nvGraphicFramePr, el:xfrm, el:a:graphic, attr:macro, attr:fPublished | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:86` | complexType | `CT_GroupShapeNonVisual` | el:cNvPr, el:cNvGrpSpPr | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:93` | complexType | `CT_GroupShape` | el:nvGrpSpPr, el:grpSpPr, el:sp, el:grpSp, el:graphicFrame, el:cxnSp, el:pic | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:106` | group | `EG_ObjectChoices` | el:sp, el:grpSp, el:graphicFrame, el:cxnSp, el:pic, el:contentPart | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:118` | complexType | `CT_Rel` | attr:r:id | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:121` | simpleType | `ST_ColID` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:126` | simpleType | `ST_RowID` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:131` | complexType | `CT_Marker` | el:col, el:colOff, el:row, el:rowOff | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:139` | simpleType | `ST_EditAs` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:146` | complexType | `CT_TwoCellAnchor` | el:from, el:to, group:EG_ObjectChoices, el:clientData, attr:editAs | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:155` | complexType | `CT_OneCellAnchor` | el:from, el:ext, group:EG_ObjectChoices, el:clientData | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:163` | complexType | `CT_AbsoluteAnchor` | el:pos, el:ext, group:EG_ObjectChoices, el:clientData | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:171` | group | `EG_Anchor` | el:twoCellAnchor, el:oneCellAnchor, el:absoluteAnchor | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:178` | complexType | `CT_Drawing` | group:EG_Anchor | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-spreadsheetDrawing.xsd:183` | element | `wsDr` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |

### dml-wordprocessingDrawing.xsd

| Anchor | Kind | Declaration | Members | Status | Evidence / gap |
|---|---|---|---|---|---|
| `dml-wordprocessingDrawing.xsd:14` | complexType | `CT_EffectExtent` | attr:l, attr:t, attr:r, attr:b | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:20` | simpleType | `ST_WrapDistance` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:23` | complexType | `CT_Inline` | el:extent, el:effectExtent, el:docPr, el:cNvGraphicFramePr, el:a:graphic, attr:distT, attr:distB, attr:distL ... | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:37` | simpleType | `ST_WrapText` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:45` | complexType | `CT_WrapPath` | el:start, el:lineTo, attr:edited | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:52` | complexType | `CT_WrapNone` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:53` | complexType | `CT_WrapSquare` | el:effectExtent, attr:wrapText, attr:distT, attr:distB, attr:distL, attr:distR | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:63` | complexType | `CT_WrapTight` | el:wrapPolygon, attr:wrapText, attr:distL, attr:distR | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:71` | complexType | `CT_WrapThrough` | el:wrapPolygon, attr:wrapText, attr:distL, attr:distR | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:79` | complexType | `CT_WrapTopBottom` | el:effectExtent, attr:distT, attr:distB | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:86` | group | `EG_WrapType` | el:wrapNone, el:wrapSquare, el:wrapTight, el:wrapThrough, el:wrapTopAndBottom | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:97` | simpleType | `ST_PositionOffset` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:100` | simpleType | `ST_AlignH` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:109` | simpleType | `ST_RelFromH` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:121` | complexType | `CT_PosH` | el:align, el:posOffset, attr:relativeFrom | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:130` | simpleType | `ST_AlignV` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:139` | simpleType | `ST_RelFromV` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:151` | complexType | `CT_PosV` | el:align, el:posOffset, attr:relativeFrom | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:160` | complexType | `CT_Anchor` | el:simplePos, el:positionH, el:positionV, el:extent, el:effectExtent, group:EG_WrapType, el:docPr, el:cNvGraphicFramePr ... | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:185` | complexType | `CT_TxbxContent` | group:w:EG_BlockLevelElts | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:188` | complexType | `CT_TextboxInfo` | el:txbxContent, el:extLst, attr:id | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:195` | complexType | `CT_LinkedTextboxInformation` | el:extLst, attr:id, attr:seq | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:202` | complexType | `CT_WordprocessingShape` | el:cNvPr, el:cNvSpPr, el:cNvCnPr, el:spPr, el:style, el:extLst, el:txbx, el:linkedTxbx ... | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:223` | complexType | `CT_GraphicFrame` | el:cNvPr, el:cNvFrPr, el:xfrm, el:a:graphic, el:extLst | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:233` | complexType | `CT_WordprocessingContentPartNonVisual` | el:cNvPr, el:cNvContentPartPr | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:239` | complexType | `CT_WordprocessingContentPart` | el:nvContentPartPr, el:xfrm, el:extLst, attr:bwMode, attr:r:id | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:248` | complexType | `CT_WordprocessingGroup` | el:cNvPr, el:cNvGrpSpPr, el:grpSpPr, el:wsp, el:grpSp, el:graphicFrame, el:dpct:pic, el:contentPart ... | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:264` | complexType | `CT_WordprocessingCanvas` | el:bg, el:whole, el:wsp, el:dpct:pic, el:contentPart, el:wgp, el:graphicFrame, el:extLst ... | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:278` | element | `wpc` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:272` | element | `wgp` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:255` | element | `wsp` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:281` | element | `inline` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |
| `dml-wordprocessingDrawing.xsd:282` | element | `anchor` | - | Out of renderer scope | Host drawing schema for Word/Spreadsheet, not a PresentationML renderer target. |

## Maintenance Rules

1. Regenerate this file after changing schema scope or coverage classification:

   ```text
   python3 tools/generate_ooxml_drawingml_audit.py
   ```

2. A declaration may move to **Supported** only with source-schema anchors,
   parser/lowering evidence, deterministic fixtures, and renderer/reporting
   tests for excluded subclauses.
3. Do not mark a declaration supported because a real-world screenshot looks
   close. Source semantics and fixture proof come first.
