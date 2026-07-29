import React from "react";
import * as GQL from "src/core/generated-graphql";
import { Modal, Button } from "react-bootstrap";

interface IEditAudiosDialog {
  selected: GQL.SlimAudioDataFragment[];
  onClose: () => void;
}

export const EditAudiosDialog: React.FC<IEditAudiosDialog> = ({
  selected,
  onClose,
}) => {
  return (
    <Modal show onHide={onClose} size="lg">
      <Modal.Header closeButton>
        <Modal.Title>Edit {selected.length} Audio(s)</Modal.Title>
      </Modal.Header>
      <Modal.Body>
        <p>Bulk edit functionality for audios will be implemented here.</p>
        <p>Selected: {selected.map((a) => a.title || "Untitled").join(", ")}</p>
      </Modal.Body>
      <Modal.Footer>
        <Button variant="secondary" onClick={onClose}>
          Cancel
        </Button>
        <Button variant="primary" onClick={onClose}>
          Apply Changes
        </Button>
      </Modal.Footer>
    </Modal>
  );
};
