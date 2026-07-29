import React from "react";
import { Form } from "react-bootstrap";
import { useIntl } from "react-intl";
import { CriterionModifier } from "../../../core/generated-graphql";
import { INumberValue } from "../../../models/list-filter/types";
import { NumberCriterion } from "../../../models/list-filter/criteria/criterion";
import { NumberField } from "src/utils/form";

interface IAudioBitrateFilterProps {
  criterion: NumberCriterion;
  onValueChanged: (value: INumberValue) => void;
}

export const AudioBitrateFilter: React.FC<IAudioBitrateFilterProps> = ({
  criterion,
  onValueChanged,
}) => {
  const intl = useIntl();

  const { value } = criterion;

  function onChanged(
    event: React.ChangeEvent<HTMLInputElement>,
    property: "value" | "value2"
  ) {
    const numericValue = parseInt(event.target.value, 10);
    const valueCopy = { ...value };

    valueCopy[property] = !Number.isNaN(numericValue) ? numericValue : 0;
    onValueChanged(valueCopy);
  }

  let equalsControl: JSX.Element | null = null;
  if (
    criterion.modifier === CriterionModifier.Equals ||
    criterion.modifier === CriterionModifier.NotEquals
  ) {
    equalsControl = (
      <Form.Group>
        <NumberField
          className="btn-secondary"
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            onChanged(e, "value")
          }
          value={value?.value ?? ""}
          placeholder={intl.formatMessage({ id: "criterion.value" })}
          min={0}
          step={1000}
        />
        <Form.Text className="text-muted">
          {intl.formatMessage({ id: "audio.bitrate_hint" }, { unit: "bps" })}
        </Form.Text>
      </Form.Group>
    );
  }

  let lowerControl: JSX.Element | null = null;
  if (
    criterion.modifier === CriterionModifier.GreaterThan ||
    criterion.modifier === CriterionModifier.Between ||
    criterion.modifier === CriterionModifier.NotBetween
  ) {
    lowerControl = (
      <Form.Group>
        <NumberField
          className="btn-secondary"
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            onChanged(e, "value")
          }
          value={value?.value ?? ""}
          placeholder={intl.formatMessage({ id: "criterion.greater_than" })}
          min={0}
          step={1000}
        />
        <Form.Text className="text-muted">
          {intl.formatMessage({ id: "audio.bitrate_hint" }, { unit: "bps" })}
        </Form.Text>
      </Form.Group>
    );
  }

  let upperControl: JSX.Element | null = null;
  if (
    criterion.modifier === CriterionModifier.LessThan ||
    criterion.modifier === CriterionModifier.Between ||
    criterion.modifier === CriterionModifier.NotBetween
  ) {
    upperControl = (
      <Form.Group>
        <NumberField
          className="btn-secondary"
          onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
            onChanged(
              e,
              criterion.modifier === CriterionModifier.LessThan
                ? "value"
                : "value2"
            )
          }
          value={
            (criterion.modifier === CriterionModifier.LessThan
              ? value?.value
              : value?.value2) ?? ""
          }
          placeholder={intl.formatMessage({ id: "criterion.less_than" })}
          min={0}
          step={1000}
        />
        <Form.Text className="text-muted">
          {intl.formatMessage({ id: "audio.bitrate_hint" }, { unit: "bps" })}
        </Form.Text>
      </Form.Group>
    );
  }

  return (
    <>
      {equalsControl}
      {lowerControl}
      {upperControl}
    </>
  );
};
