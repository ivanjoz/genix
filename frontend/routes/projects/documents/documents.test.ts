import { describe, expect, test } from 'bun:test';
import type { ISprint, IStory } from '../project.ts';
import { projectDocuments, storyDocumentGroups, type IDocument } from './documents.ts';

const makeDocument = (key: string, storyID: number, uploadedAt: number, ss = 1): IDocument =>
  ({ ProjectID: 1, StoryID: storyID, Key: key, Name: key, ContentType: 'text/plain', Size: 10, UploadedAt: uploadedAt, ss })

const makeStory = (id: number, number: number, lastSprintID = 0, ss = 1): IStory =>
  ({ ProjectID: 1, ID: id, Type: 1, Title: `Story ${number}`, Priority: 2, BacklogRank: number, State: 0, LastSprintID: lastSprintID, ss })

describe('projectDocuments', () => {
  test('keeps the active documents without a story, newest first', () => {
    const documents = [makeDocument('old', 0, 10), makeDocument('story', 4, 50), makeDocument('new', 0, 30), makeDocument('deleted', 0, 40, 0)]
    expect(projectDocuments(documents).map((document) => document.Key)).toEqual(['new', 'old'])
  })
})

describe('storyDocumentGroups', () => {
  const sprintsByID = new Map([[7, { ID: 7, Name: 'Sprint 2' } as ISprint]])

  test('groups the files by story, newest story first, labelled by its last sprint', () => {
    const storiesByID = new Map([[1, makeStory(1, 1, 7)], [2, makeStory(2, 2)]])
    const documents = [makeDocument('a', 1, 10), makeDocument('b', 2, 20), makeDocument('c', 1, 30), makeDocument('project', 0, 40)]
    const groups = storyDocumentGroups(documents, storiesByID, sprintsByID)
    expect(groups.map((group) => [group.story.ID, group.sprintLabel, group.documents.map((document) => document.Key)])).toEqual([
      [2, 'Backlog|Backlog', ['b']],
      [1, 'Sprint 2', ['c', 'a']],
    ])
  })

  test('leaves out deleted files and the files of deleted or unknown stories', () => {
    const storiesByID = new Map([[1, makeStory(1, 1)], [2, makeStory(2, 2, 0, 0)]])
    const documents = [makeDocument('deleted', 1, 10, 0), makeDocument('of-deleted-story', 2, 20), makeDocument('of-unknown-story', 9, 30)]
    expect(storyDocumentGroups(documents, storiesByID, sprintsByID)).toEqual([])
  })
})
